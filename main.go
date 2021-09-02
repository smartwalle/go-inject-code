package main

import (
	"flag"
	"go-inject-code/internal"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

func main() {
	var input string
	flag.StringVar(&input, "input", "", "指定 go go 源代码文件所在目录")
	flag.Parse()

	if len(input) == 0 {
		log.Fatal("需要指定 go 源代码文件所在目录，如: --input \"./\"")
		return
	}

	filepath.Walk(input, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
			return err
		}

		if info.IsDir() {
			return err
		}

		if !strings.HasSuffix(strings.ToLower(info.Name()), ".go") {
			return nil
		}

		var areas []internal.TextArea
		areas, err = internal.Load(path)
		if err != nil {
			log.Fatal(err)
			return err
		}

		if err = internal.Write(path, areas); err != nil {
			log.Fatal(err)
		}
		return nil
	})
}
