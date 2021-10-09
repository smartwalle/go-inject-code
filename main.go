package main

import (
	"flag"
	"github.com/smartwalle/go-inject-code/internal"
	"github.com/smartwalle/go-inject-code/internal/inject_field"
	"github.com/smartwalle/go-inject-code/internal/inject_import"
	"github.com/smartwalle/go-inject-code/internal/inject_tag"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

func main() {
	var input string
	var file string
	var tag string
	flag.StringVar(&input, "input", "", "指定 go 源代码文件所在目录，如：--input=\"./\"")
	flag.StringVar(&file, "file", "", "指定 go 源代码文件，多个文件使用 '|' 进行分割，如：--file=\"./test.go\"")
	flag.StringVar(&tag, "tag", "", "自动生成 tag, 多个 tag 使用 '|' 进行分割，如： --tag=\"sql|bson\"")
	flag.Parse()

	// 清理参数
	input = strings.TrimSpace(input)
	file = strings.TrimSpace(file)

	if input == "" && file == "" {
		log.Fatal("需要指定 go 源代码文件所在目录，如: --input=\"./\"")
		return
	}

	internal.RegisterFieldProcessor(inject_tag.NewProcessField(tag))
	internal.RegisterStructProcessor(inject_field.NewProcessStruct())
	internal.RegisterImportProcessor(inject_import.NewProcessImport())

	// 处理目录
	if input != "" {
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

			return parse(path)
		})
	}

	// 处理文件
	if file != "" {
		var files = strings.Split(file, "|")
		for _, path := range files {
			if strings.HasSuffix(strings.ToLower(path), ".go") {
				parse(path)
			}
		}
	}
}

func parse(path string) (err error) {
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
}
