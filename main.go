package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func ListFiles(rootPath string) ([]string, error) {
	paths := make([]string, 0)

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}

		paths = append(paths, relativePath)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run main.go <directory>")
		os.Exit(1)
	}

	files, err := ListFiles(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, file := range files {
		fmt.Println(file)
	}
}
