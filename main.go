package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/alitto/pond"
)

type Stats struct {
	files []*File
	mutex sync.Mutex
}

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
	parent *Folder
	name   string
	size   uint64
}

func main() {
	var wg sync.WaitGroup
	pool := pond.New(runtime.NumCPU()*10, 1000000)
	baseDir := Folder{name: "/"}
	stats := Stats{files: []*File{}}

	wg.Add(1)
	pool.Submit(func() {
		walk("/", &baseDir, &stats, &wg, pool)
	})

	wg.Wait()
	pool.Stop()

	fmt.Println("STATS", "Size:", baseDir.size, "Files:", baseDir.fileCount, "Folders:", baseDir.folderCount)

	sort.Slice(stats.files, func(i, j int) bool {
		return stats.files[i].size > stats.files[j].size
	})

	fmt.Println("TOP 10 LARGEST FILES")
	for i := 0; i < 10; i++ {
		file := stats.files[i]
		fmt.Println(file.parent.name+"/"+file.name, file.size)
	}
}

func walk(dir string, folder *Folder, stats *Stats, wg *sync.WaitGroup, pool *pond.WorkerPool) {
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
			nextFolder := Folder{name: f.Name(), parent: folder}
			folder.folders = append(folder.folders, &nextFolder)
			folderCount++

			wg.Add(1)
			pool.Submit(func() {
				walk(filepath.Join(dir, nextFolder.name), &nextFolder, stats, wg, pool)
			})
		} else {
			info, _ := f.Info()
			file := File{folder, f.Name(), uint64(info.Size())}
			folder.files = append(folder.files, &file)
			stats.mutex.Lock()
			stats.files = append(stats.files, &file)
			stats.mutex.Unlock()
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
