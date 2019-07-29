package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	workdir = ".cover"
	profile = workdir + "/cover.out"
	mode    = "count"
	runHtml bool
)

func main() {

	flag.BoolVar(&runHtml, "html", false, "show html coverage report")
	flag.Parse()

	generateCoverData()

	runCover("func")

	if runHtml {
		runCover("html")
	}
}

func generateCoverData() {
	err := os.RemoveAll(workdir)
	if err != nil {
		log.Fatal("error deleting workdir: ", err)
	}
	err = os.Mkdir(workdir, os.FileMode(int(0777)))
	if err != nil {
		log.Fatal("error creating workdir: ", err)
	}
	pkgs := getPackages()

	var wg sync.WaitGroup

	for _, pkg := range pkgs {
		wg.Add(1)
		go func(pkg string) {
			defer wg.Done()
			runTestsInDir(pkg)
		}(pkg)
	}

	wg.Wait()

	file, err := os.Create(profile)
	if err != nil {
		log.Fatal("error creating profile file: ", err)
	}
	defer file.Close()

	_, err = file.WriteString("mode: count\n")
	if err != nil {
		log.Fatal("error writing to profile file: ", err)
	}

	//todo: append *.cover files to profile file
	wd, err := os.Open(workdir)
	if err != nil {
		log.Fatal("could not open workdir: ", err)
	}
	defer wd.Close()
	files, err := wd.Readdirnames(0)
	if err != nil {
		log.Fatal("error getting file names: ", err)
	}
	for _, coverFile := range files {
		if strings.HasSuffix(coverFile, ".cover") {
			f, err := os.Open(fmt.Sprintf("%s/%s", workdir, coverFile))
			defer f.Close()
			if err != nil {
				log.Fatal("couldn't open ", coverFile, ": ", err)
			}

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				text := scanner.Text()
				if text == "mode: count" {
					continue
				}

				_, err = io.Copy(file, strings.NewReader(text+"\n"))
				if err != nil {
					log.Print("error writing to profile: ", err)
				}
			}
		}
	}

}

func runTestsInDir(dir string) {
	f := strings.Replace(dir, "/", "-", -1)
	f = fmt.Sprintf("%s/%s.cover", workdir, f)

	cmd := exec.Command("go", "test", "-covermode=count", fmt.Sprintf("-coverprofile=%s", f), dir)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Print("err attaching to stdout:", err)
	}

	done := make(chan struct{})
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		done <- struct{}{}
	}()

	errReader, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("err attaching to stderr: %+v", err)
	}
	errDone := make(chan struct{})
	errScanner := bufio.NewScanner(errReader)
	go func() {
		for errScanner.Scan() {
			fmt.Println(errScanner.Text())
		}
		errDone <- struct{}{}
	}()

	err = cmd.Start()
	if err != nil {
		log.Printf("err starting cmd: %+v", err)
	}

	<-done
	<-errDone

	err = cmd.Wait()
	if err != nil {
		log.Printf("err cmd.wait: %+v", err)
	}
}

func runCover(param string) {
	cmd := exec.Command("go", "tool", "cover", fmt.Sprintf("--%s=%s", param, profile))
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("err attaching to stdout: %+v", err)
	}
	done := make(chan struct{})
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		done <- struct{}{}
	}()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("err attaching to stderr: %+v", err)
	}

	errDone := make(chan struct{})
	errScanner := bufio.NewScanner(stderr)
	go func() {
		for errScanner.Scan() {
			fmt.Println(errScanner.Text())
		}
		errDone <- struct{}{}
	}()

	err = cmd.Start()
	if err != nil {
		log.Printf("err starting cmd: %+v", err)
	}

	<-done
	<-errDone
	err = cmd.Wait()
	if err != nil {
		log.Printf("err after wait(--%s=%s): %v", param, profile, err)
	}

}

func getPackages() []string {
	cmd := exec.Command("go", "list", "./...")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("stdout: %+v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("err attaching to stderr: %+v", err)
	}

	err = cmd.Start()
	if err != nil {
		log.Printf("cmd start: %+v", err)
	}

	slurp, err := ioutil.ReadAll(stderr)
	if err != nil {
		log.Printf("err reading stderr: %+v", err)
	}

	lines := []string{}
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("getPackages scanner: %v", err)
	}

	err = cmd.Wait()
	if err != nil {
		switch err := err.(type) {
		case *exec.ExitError:
			log.Printf("stderr from `go list`:\n%s", string(slurp))
		default:
			log.Printf("wait: %v", err)
		}

	}

	/* 	out, err := exec.Command("go", "list", "./...").Output()
	   	if err != nil {
	   		log.Fatal("getPackages cmd:", err)
	   	} */

	return lines
}
