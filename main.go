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

	internal.RegisterTagGenerator(inject_tag.NewTagGenerator(tag))
	internal.RegisterFieldGenerator(inject_field.NewFieldGenerator())
	internal.RegisterImportGenerator(inject_import.NewImportGenerator())

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

			return parse(path)
		})
	}

	// 处理文件
	if file != "" {
		var filenames = strings.Split(file, "|")
		for _, filename := range filenames {
			parse(filename)
		}
	}
}

func parse(filename string) (err error) {
	if !strings.HasSuffix(strings.ToLower(filename), ".go") {
		return nil
	}

	var areas []internal.TextArea
	areas, err = internal.Load(filename)
	if err != nil {
		log.Fatal(err)
		return err
	}

	if err = internal.Write(filename, areas); err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}
