package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

func HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err = io.Copy(hasher, file); err != nil {
		return "", err
	}

	result := hasher.Sum(nil)
	hexString := hex.EncodeToString(result)

	return hexString, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run main.go <directory>")
		os.Exit(1)
	}

	rootPath := os.Args[1]

	files, err := ListFiles(rootPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, file := range files {
		fullPath := filepath.Join(rootPath, file)

		hash, err := HashFile(fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to hash %q: %v\n", file, err)

			continue
		}

		fmt.Printf("%s: %s\n", file, hash)
	}
}
