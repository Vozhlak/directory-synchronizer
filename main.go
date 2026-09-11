package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

const NumWorkers = 10

type FileHash struct {
	Path string
	Hash string
}

type scanResult struct {
	m   map[string]string
	err error
}

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

func ScanDir(rootPath string) (map[string]string, error) {
	files, err := ListFiles(rootPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	pathAndHashes := make(map[string]string)

	tasks := make(chan string, len(files))
	results := make(chan FileHash, len(files))

	wg := sync.WaitGroup{}

	wg.Add(NumWorkers)
	for i := 0; i < NumWorkers; i++ {

		go func() {
			defer wg.Done()

			for path := range tasks {
				hash, err := HashFile(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to hash %q: %v\n", path, err)

					continue
				}

				results <- FileHash{
					Path: path,
					Hash: hash,
				}
			}
		}()
	}

	for _, file := range files {
		fullPath := filepath.Join(rootPath, file)

		tasks <- fullPath
	}

	close(tasks)

	go func() {
		wg.Wait()
		close(results)
	}()

	for fh := range results {
		relPath, err := filepath.Rel(rootPath, fh.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to get relative path: %v\n", err)
			continue
		}

		pathAndHashes[relPath] = fh.Hash
	}

	return pathAndHashes, nil
}

func CompareScans(source, dest map[string]string) (toCopy, toUpdate, toDelete []string) {
	toCopy = make([]string, 0, len(source))
	toUpdate = make([]string, 0, len(source))
	toDelete = make([]string, 0, len(source))

	for path, srcHash := range source {
		if destHash, exists := dest[path]; !exists {
			toCopy = append(toCopy, path)
		} else if srcHash != destHash {
			toUpdate = append(toUpdate, path)
		}
	}

	for destPath := range dest {
		if _, exists := source[destPath]; !exists {
			toDelete = append(toDelete, destPath)
		}
	}

	return toCopy, toUpdate, toDelete
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: go run main.go <source> <dest>")
		os.Exit(1)
	}

	sourcePath := os.Args[1]
	destPath := os.Args[2]

	srcCh := make(chan scanResult, 1)
	dstCh := make(chan scanResult, 1)

	go func() {
		m, err := ScanDir(sourcePath)
		srcCh <- scanResult{m, err}
	}()

	go func() {
		m, err := ScanDir(destPath)
		dstCh <- scanResult{m, err}
	}()

	sourceMap := <-srcCh
	destMap := <-dstCh

	if sourceMap.err != nil {
		fmt.Fprintf(os.Stderr, "error scanning source: %v\n", sourceMap.err)
		os.Exit(1)
	}

	if destMap.err != nil {
		fmt.Fprintf(os.Stderr, "error scanning dest: %v\n", destMap.err)
		os.Exit(1)
	}

	toCopy, toUpdate, toDelete := CompareScans(sourceMap.m, destMap.m)

	fmt.Println("Файлы для КОПИРОВАНИЯ:")
	for _, toCopyItem := range toCopy {
		fmt.Printf("- %s\n", toCopyItem)
	}

	fmt.Println("Файлы для ОБНОВЛЕНИЯ:")
	for _, toUpdateItem := range toUpdate {
		fmt.Printf("- %s\n", toUpdateItem)
	}

	fmt.Println("Файлы для УДАЛЕНИЯ:")
	for _, toDeleteItem := range toDelete {
		fmt.Printf("- %s\n", toDeleteItem)
	}
}
