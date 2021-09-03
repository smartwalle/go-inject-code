package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
)

type FieldProcessor func(field *ast.Field, comments []*ast.Comment) TextArea

var (
	fieldProcessors = make([]FieldProcessor, 0, 1)
)

func RegisterFieldProcessor(p FieldProcessor) {
	if p == nil {
		return
	}
	fieldProcessors = append(fieldProcessors, p)
}

type TextArea interface {
	Inject(content []byte) []byte
}

func Load(path string) (areas []TextArea, err error) {
	var fileSet = token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		var typeSpec *ast.TypeSpec
		for _, spec := range genDecl.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				typeSpec = ts
				break
			}
		}

		if typeSpec == nil {
			continue
		}

		structDecl, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		for _, field := range structDecl.Fields.List {
			var comments = make([]*ast.Comment, 0, 2)

			if field.Doc != nil {
				comments = append(comments, field.Doc.List...)
			}

			if field.Comment != nil {
				comments = append(comments, field.Comment.List...)
			}

			for _, p := range fieldProcessors {
				var nArea = p(field, comments)
				if nArea != nil {
					areas = append(areas, nArea)
				}
			}
		}
	}
	return
}

func Write(path string, areas []TextArea) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	if err = file.Close(); err != nil {
		return err
	}

	for i := range areas {
		area := areas[len(areas)-i-1]
		content = area.Inject(content)
	}

	if err = ioutil.WriteFile(path, content, 0644); err != nil {
		return err
	}

	return
}
