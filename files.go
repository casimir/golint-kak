package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func copyFileContents(src, dst string) (err error) {
	fin, err := os.Open(src)
	if err != nil {
		return
	}
	defer fin.Close()
	fout, err := os.Create(dst)
	if err != nil {
		return
	}
	if _, err = io.Copy(fout, fin); err != nil {
		return
	}
	return fout.Close()
}

type fileProvider struct {
	Dir       string
	Filenames []string
	TempFiles map[string]string
}

func newFileProvider(names ...string) (*fileProvider, error) {
	fp := fileProvider{TempFiles: map[string]string{}}
	for _, it := range names {
		if filepath.Ext(it) == ".go" {
			fp.Filenames = append(fp.Filenames, it)
			continue
		}
		dir, err := fp.TempDir()
		if err != nil {
			return nil, err
		}
		tempFile := filepath.Join(dir, filepath.Base(it)+".go")
		if err := copyFileContents(it, tempFile); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create temporary file for %q: %s", it, err)
			continue
		}
		fp.Filenames = append(fp.Filenames, tempFile)
		fp.TempFiles[tempFile] = it

	}
	return &fp, nil
}

func (fp *fileProvider) TempDir() (string, error) {
	if fp.Dir == "" {
		tmp, err := ioutil.TempDir("", "golint-kak.")
		if err != nil {
			return "", err
		}
		fp.Dir = tmp
	}
	return fp.Dir, nil
}

func (fp fileProvider) Clean() {
	if fp.Dir != "" {
		os.RemoveAll(fp.Dir)
	}
}

func (fp fileProvider) Filename(file string) string {
	if real, ok := fp.TempFiles[file]; ok {
		return real
	}
	return file
}
