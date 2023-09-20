package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"regexp"
	"strings"
)

type TagGenerator func(f *ast.Field) TextArea
type FieldGenerator func(s *ast.StructType, comments []*ast.Comment) TextArea
type ImportGenerator func(f *ast.File) TextArea

var (
	tagGenerator    TagGenerator
	fieldGenerator  FieldGenerator
	importGenerator ImportGenerator
)

func RegisterTagGenerator(p TagGenerator) {
	tagGenerator = p
}

func RegisterFieldGenerator(p FieldGenerator) {
	fieldGenerator = p
}

func RegisterImportGenerator(p ImportGenerator) {
	importGenerator = p
}

type TextArea interface {
	Inject(content []byte) []byte
}

func Load(filename string) (areas []TextArea, err error) {
	var fileSet = token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	if importGenerator != nil {
		var nArea = importGenerator(file)
		if nArea != nil {
			areas = append(areas, nArea)
		}
	}

	for _, dec := range file.Decls {
		var genDec, ok = dec.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDec.Specs {
			var nAreas []TextArea
			switch rSpec := spec.(type) {
			case *ast.TypeSpec:
				nAreas = parseType(genDec, rSpec)
			}
			areas = append(areas, nAreas...)
		}
	}
	return
}

func parseType(genDec *ast.GenDecl, rSpec *ast.TypeSpec) []TextArea {
	switch rType := rSpec.Type.(type) {
	case *ast.StructType:
		return parseStruct(genDec, rType)
	}
	return nil
}

func parseStruct(genDec *ast.GenDecl, structType *ast.StructType) (areas []TextArea) {
	if structType == nil {
		return nil
	}

	for _, field := range structType.Fields.List {
		if tagGenerator != nil {
			var nArea = tagGenerator(field)
			if nArea != nil {
				areas = append(areas, nArea)
			}
		}
	}

	if fieldGenerator != nil && genDec.Doc != nil {
		var nArea = fieldGenerator(structType, genDec.Doc.List)
		if nArea != nil {
			areas = append(areas, nArea)
		}
	}
	return areas
}

func Write(filename string, areas []TextArea) (err error) {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	content, err := io.ReadAll(file)
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

	if err = os.WriteFile(filename, content, 0644); err != nil {
		return err
	}

	return
}

var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func SnakeCase(str string) string {
	snake := matchAllCap.ReplaceAllString(str, "${1}_${2}")
	return strings.ToLower(snake)
}
