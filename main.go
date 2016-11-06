package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/lint"
)

var (
	minConfidence = flag.Float64("min_confidence", 0.8, "minimum confidence of a problem to print it")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\tgolintfile [flags] files... # must be a single package\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	files, err := newFileProvider(flag.Args()...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temporary directory: %s", err)
		os.Exit(1)
	}
	defer files.Clean()
	lintFiles(files)
	vetFiles(files)
}

func lintFiles(provider *fileProvider) {
	files := make(map[string][]byte)
	for _, filename := range provider.Filenames {
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		files[filename] = src
	}

	l := new(lint.Linter)
	ps, err := l.LintFiles(files)
	if err != nil {
		// this is a parse error from go/parser vetFiles will report it
		return
	}
	for _, p := range ps {
		if p.Confidence >= *minConfidence {
			file := provider.Filename(p.Position.Filename)
			line := p.Position.Line
			column := p.Position.Column
			fmt.Printf("%s:%d:%d: warning: %s (golint)\n", file, line, column, p.Text)
		}
	}
}

func vetFiles(provider *fileProvider) {
	if len(provider.Filenames) == 0 {
		return
	}
	args := []string{"tool", "vet"}
	c := exec.Command("go", append(args, provider.Filenames...)...)
	// go vet returns 1 when there is a match so we ignore status here
	out, _ := c.CombinedOutput()
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "vet:") {
			parts := strings.SplitN(scanner.Text(), ":", 3)
			file := parts[0]
			line := parts[1]
			message := strings.TrimSpace(parts[2])
			fmt.Printf("%s:%s:1: warning: %s (go vet)\n", provider.Filename(file), line, message)
		} else if scanner.Text() != "vet: no files checked" {
			parts := strings.SplitN(scanner.Text(), ":", 6)
			file := strings.TrimSpace(parts[2])
			line := parts[3]
			column := parts[4]
			message := strings.TrimSpace(parts[5])
			fmt.Printf("%s:%s:%s: error: %s\n", provider.Filename(file), line, column, message)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error processing `go vet` output: %v", err)
	}
}
