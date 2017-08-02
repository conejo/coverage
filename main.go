package main

import "os"

import "os/exec"
import "log"
import "bufio"
import "bytes"
import "strings"
import "fmt"

var (
	workdir = ".cover"
	profile = workdir + "cover.out"
	mode    = "count"
)

func main() {
	generateCoverData()
}

func generateCoverData() {
	_ = os.Remove(workdir)
	_ = os.Mkdir(workdir, os.FileMode(int(0777)))
	pkgs := getPackages()

	for _, pkg := range pkgs {
		runTestsInDir(pkg)
	}
	//todo: create profile file
	//todo: add 'mode=count' header text
	//todo: append *.cover files to profile file
	//todo: go tool cover --func=./.cover/api.
	//todo: handle html flag
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
