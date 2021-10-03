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

type FieldProcessor func(f *ast.Field) TextArea
type StructProcessor func(s *ast.StructType, comments []*ast.Comment) TextArea
type ImportProcessor func(f *ast.File) TextArea

var (
	fieldProcessor  FieldProcessor
	structProcessor StructProcessor
	importProcessor ImportProcessor
)

func RegisterFieldProcessor(p FieldProcessor) {
	fieldProcessor = p
}

func RegisterStructProcessor(p StructProcessor) {
	structProcessor = p
}

func RegisterImportProcessor(p ImportProcessor) {
	importProcessor = p
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

	if importProcessor != nil {
		var nArea = importProcessor(file)
		if nArea != nil {
			areas = append(areas, nArea)
		}
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
		if fieldProcessor != nil {
			var nArea = fieldProcessor(field)
			if nArea != nil {
				areas = append(areas, nArea)
			}
		}
	}

	if structProcessor != nil && genDecl.Doc != nil {
		var nArea = structProcessor(structType, genDecl.Doc.List)
		if nArea != nil {
			areas = append(areas, nArea)
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
