package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

func excludeFiles(files []os.DirEntry) []os.DirEntry {
	dirs := make([]os.DirEntry, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			dirs = append(dirs, f)
		}
	}
	return dirs
}

func dirTreeWalk(writer io.Writer, path string, printFiles bool, prefix string) error {
	content, _ := os.ReadDir(path)

	if !printFiles {
		content = excludeFiles(content)
	}

	for i, f := range content {
		indent := "├───"
		stick := "│"
		if i == len(content)-1 {
			indent = "└───"
			stick = ""
		}

		if f.IsDir() {
			_, err := fmt.Fprintf(writer, prefix+indent+"%s\n", f.Name())
			if err != nil {
				return nil
			}
			err = dirTreeWalk(writer, path+"/"+f.Name(), printFiles, prefix+stick+"\t")
			if err != nil {
				return err
			}
		} else if printFiles {
			inf, _ := f.Info()
			size := "empty"
			if inf.Size() != 0 {
				size = strconv.Itoa(int(inf.Size())) + "b"
			}
			_, err := fmt.Fprintf(writer, prefix+indent+"%s (%s)\n", f.Name(), size)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func dirTree(writer io.Writer, path string, printFiles bool) error {
	return dirTreeWalk(writer, path, printFiles, "")
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
