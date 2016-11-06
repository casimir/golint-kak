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
	lintFiles(flag.Args()...)
	vetFiles(flag.Args()...)
}

func lintFiles(filenames ...string) {
	files := make(map[string][]byte)
	for _, filename := range filenames {
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
			fmt.Printf("%v: warning: %s (golint)\n", p.Position, p.Text)
		}
	}
}

func vetFiles(filenames ...string) {
	if len(filenames) == 0 {
		return
	}
	args := []string{"tool", "vet"}
	c := exec.Command("go", append(args, filenames...)...)
	// go vet returns 1 when there is a match so we ignore status here
	out, _ := c.CombinedOutput()
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "vet:") {
			parts := strings.SplitN(scanner.Text(), ":", 3)
			file := parts[0]
			line := parts[1]
			message := strings.TrimSpace(parts[2])
			fmt.Printf("%s:%s:1: error: %s (go vet)\n", file, line, message)
		} else if scanner.Text() != "vet: no files checked" {
			parts := strings.SplitN(scanner.Text(), ":", 6)
			file := strings.TrimSpace(parts[2])
			line := parts[3]
			column := parts[4]
			message := strings.TrimSpace(parts[5])
			fmt.Printf("%s:%s:%s: warning: %s\n", file, line, column, message)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error processing `go vet` output: %v", err)
	}
}
