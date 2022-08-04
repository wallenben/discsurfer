package main

import (
	"fmt"
	"os"
	"path/filepath"
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

type WorkerPool interface {
	Run()
	AddTask(task func())
}

type workerPool struct {
	workerAmount int
	queuedTasks  chan func()
}

var baseDir = Folder{}

func (wp *workerPool) Run() {
	for i := 0; i < wp.workerAmount; i++ {
		go func(workerID int) {
			for task := range wp.queuedTasks {
				task()
			}
		}(i + 1)
	}
}
func (wp *workerPool) AddTask(task func()) {
	wp.queuedTasks <- task
}
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

	files, err := os.ReadDir(dir)

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
			go walk(filepath.Join(dir, f.Name()), &nextFolder, wg)
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
