package main

import (
	"flag"
	"github.com/smartwalle/go-inject-code/internal"
	"github.com/smartwalle/go-inject-code/internal/injectfield"
	"github.com/smartwalle/go-inject-code/internal/injectimport"
	"github.com/smartwalle/go-inject-code/internal/injecttag"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

func main() {
	var vFilepath string
	var vFilename string
	var vTag string
	flag.StringVar(&vFilepath, "filepath", "", "指定 Go 源代码文件所在目录，如：--filepath=\"./\"")
	flag.StringVar(&vFilename, "filename", "", "指定 Go 源代码文件，多个文件使用 '|' 进行分割，如：--filename=\"./test.go\"")
	flag.StringVar(&vTag, "tag", "", "自动生成 tag, 多个 tag 使用 '|' 进行分割，如： --tag=\"sql|bson\"")
	flag.Parse()

	// 清理参数
	vFilepath = strings.TrimSpace(vFilepath)
	vFilename = strings.TrimSpace(vFilename)

	if vFilepath == "" && vFilename == "" {
		log.Fatal("需要指定 Go 源代码文件所在目录，如: --input=\"./\"")
		return
	}

	internal.RegisterProcessor(injectimport.NewImportGenerator())
	internal.RegisterProcessor(injectfield.NewFieldGenerator())
	internal.RegisterProcessor(injecttag.NewTagGenerator(vTag))

	// 处理目录
	if vFilepath != "" {
		filepath.Walk(vFilepath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				log.Fatal(err)
				return err
			}

			if info.IsDir() {
				return err
			}

			return process(path)
		})
	}

	// 处理文件
	if vFilename != "" {
		var filenames = strings.Split(vFilename, "|")
		for _, filename := range filenames {
			if err := process(filename); err != nil {
				return
			}
		}
	}
}

func process(filename string) (err error) {
	if !strings.HasSuffix(strings.ToLower(filename), ".go") {
		return nil
	}

	if err = internal.Process(filename); err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}
