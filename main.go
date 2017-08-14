package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
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
					log.Fatal("error writing to profile: ", err)
				}
			}
		}
	}

}

func runTestsInDir(dir string) {
	f := dir
	if strings.Contains(dir, "/") {
		el := strings.Split(dir, "/")
		f = el[len(el)-1]
	}

	f = fmt.Sprintf("%s/%s.cover", workdir, f)

	cmd := exec.Command("go", "test", "-covermode=count", fmt.Sprintf("-coverprofile=%s", f), dir)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
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
		log.Fatal("err")
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
		log.Fatal(err)
	}

	<-done
	<-errDone

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func runCover(param string) {
	cmd := exec.Command("go", "tool", "cover", fmt.Sprintf("--%s=%s", param, profile))
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	done := make(chan struct{})
	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		done <- struct{}{}
	}()

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	<-done
	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

}

func getPackages() []string {
	out, err := exec.Command("go", "list", "./...").Output()
	if err != nil {
		log.Fatal(err)
	}

	lines := []string{}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return lines
}
