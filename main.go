package main

import (
	"fmt"
	"io/ioutil"
	"sync"
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

var baseDir = Folder{}

func main() {
	var wg sync.WaitGroup

	wg.Add(1)
	go walk(".", &baseDir, &wg)

	wg.Wait()

	fmt.Println("STATS", "Size:", baseDir.size, "Files:", baseDir.fileCount, "Folders:", baseDir.folderCount)
}

func walk(dir string, folder *Folder, wg *sync.WaitGroup) {
	defer wg.Done()
	folder.name = dir
	folder.folders = []*Folder{}
	folder.files = []*File{}

	files, err := ioutil.ReadDir(dir)

	if err != nil {
		return
	}

	folderCount, fileCount, size := uint32(0), uint32(0), uint64(0)

	for _, f := range files {
		if f.IsDir() {
			nextFolder := Folder{}
			nextFolder.parent = folder
			folder.folders = append(folder.folders, &nextFolder)
			folderCount++

			wg.Add(1)
			go walk(dir+"/"+f.Name(), &nextFolder, wg)
		} else {
			file := File{f.Name(), uint64(f.Size())}
			folder.files = append(folder.files, &file)
			size += uint64(f.Size())
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
