package main

import (
	"flag"
	"go-inject-code/internal"
	"go-inject-code/internal/inject_tag"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

func main() {
	var input string
	var tag string
	flag.StringVar(&input, "input", "", "指定 go go 源代码文件所在目录，如：--input \"./\"")
	flag.StringVar(&tag, "tag", "", "自动生成 tag, 多个 tag 使用 '|' 进行分割，如： --tag \"sql|bson\"")
	flag.Parse()

	if len(input) == 0 {
		log.Fatal("需要指定 go 源代码文件所在目录，如: --input \"./\"")
		return
	}

	internal.RegisterFieldProcessor(inject_tag.NewProcessField(strings.Split(tag, "|")))

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
			return err
		}
		return nil
	})
}
