package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

type FieldProcessor func(f *ast.Field, comments []*ast.Comment) TextArea
type StructProcessor func(s *ast.StructType, comments []*ast.Comment) TextArea

var (
	fieldProcessors  = make([]FieldProcessor, 0, 1)
	structProcessors = make([]StructProcessor, 0, 1)
)

func RegisterFieldProcessor(p FieldProcessor) {
	if p == nil {
		return
	}
	fieldProcessors = append(fieldProcessors, p)
}

func RegisterStructProcessor(p StructProcessor) {
	if p == nil {
		return
	}
	structProcessors = append(structProcessors, p)
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
		var genDecl, _ = decl.(*ast.GenDecl)
		if genDecl == nil {
			continue
		}

		for _, spec := range genDecl.Specs {
			var nAreas []TextArea
			switch rSpec := spec.(type) {
			case *ast.TypeSpec:
				nAreas = parseType(genDecl, rSpec)
			}
			areas = append(areas, nAreas...)
		}
	}
	return
}

func parseType(genDecl *ast.GenDecl, rSpec *ast.TypeSpec) []TextArea {
	switch rType := rSpec.Type.(type) {
	case *ast.StructType:
		return parseStruct(genDecl, rType)
	}
	return nil
}

func parseStruct(genDecl *ast.GenDecl, structType *ast.StructType) (areas []TextArea) {
	if structType == nil {
		return nil
	}

	for _, field := range structType.Fields.List {
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

	for _, p := range structProcessors {
		if genDecl.Doc != nil {
			var nArea = p(structType, genDecl.Doc.List)
			if nArea != nil {
				areas = append(areas, nArea)
			}
		}
	}
	return areas
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

var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func SnakeCase(str string) string {
	snake := matchAllCap.ReplaceAllString(str, "${1}_${2}")
	return strings.ToLower(snake)
}
