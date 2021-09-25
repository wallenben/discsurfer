package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/alitto/pond"
)

type Folder struct {
	name        string
	folders     []*Folder
	files       []*File
	folderCount uint32
	fileCount   uint32
	size        uint64
	parent      *Folder
	mutex       sync.Mutex
}

type File struct {
	name string
	size uint64
}

func main() {
	var wg sync.WaitGroup
	pool := pond.New(5000, 1000000)
	baseDir := Folder{name: "/"}

	wg.Add(1)
	pool.Submit(func() {
		walk(".", &baseDir, &wg, pool)
	})

	wg.Wait()
	pool.Stop()

	fmt.Println("STATS", "Size:", baseDir.size, "Files:", baseDir.fileCount, "Folders:", baseDir.folderCount)
}

func walk(dir string, folder *Folder, wg *sync.WaitGroup, pool *pond.WorkerPool) {
	defer wg.Done()
	folder.folders = []*Folder{}
	folder.files = []*File{}

	files, err := os.ReadDir(dir)

	if err != nil {
		return
	}

	folderCount, fileCount, size := uint32(0), uint32(0), uint64(0)

	for _, f := range files {
		if f.IsDir() {
			nextFolder := Folder{name: f.Name()}
			nextFolder.parent = folder
			folder.folders = append(folder.folders, &nextFolder)
			folderCount++

			wg.Add(1)
			pool.Submit(func() {
				walk(filepath.Join(dir, nextFolder.name), &nextFolder, wg, pool)
			})
		} else {
			info, _ := f.Info()
			file := File{f.Name(), uint64(info.Size())}
			folder.files = append(folder.files, &file)
			size += uint64(info.Size())
			fileCount++
		}
	}

	folder.mutex.Lock()
	folder.folderCount += folderCount
	folder.fileCount += fileCount
	folder.size += size
	folder.mutex.Unlock()

	parent := folder.parent

	for parent != nil {
		parent.mutex.Lock()
		parent.folderCount += folderCount
		parent.fileCount += fileCount
		parent.size += size
		parent.mutex.Unlock()
		parent = parent.parent
	}
}
