package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

type TextArea interface {
	Inject(content []byte) []byte
}

type Processor interface {
	File(file *ast.File) TextArea

	Struct(structType *ast.StructType, comments []*ast.Comment) TextArea

	Field(field *ast.Field) TextArea
}

var processors []Processor

func RegisterProcessor(p Processor) {
	processors = append(processors, p)
}

func Process(filename string) error {
	var content, err = os.ReadFile(filename)
	if err != nil {
		return err
	}

	for _, p := range processors {
		content, err = process(p, content)
		if err != nil {
			return err
		}
	}

	if err = os.WriteFile(filename, content, 0644); err != nil {
		return err
	}
	return nil
}

func process(p Processor, content []byte) ([]byte, error) {
	var set = token.NewFileSet()
	file, err := parser.ParseFile(set, "", content, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var areas []TextArea

	var area = p.File(file)
	if area != nil {
		areas = append(areas, area)
	}

	for _, dec := range file.Decls {
		var genDec, ok = dec.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDec.Specs {
			var nAreas []TextArea
			switch tSpec := spec.(type) {
			case *ast.TypeSpec:
				nAreas = processType(p, genDec, tSpec)
			}
			areas = append(areas, nAreas...)
		}
	}

	for i := range areas {
		area = areas[len(areas)-i-1]
		content = area.Inject(content)
	}

	return content, nil
}

func processType(p Processor, dec *ast.GenDecl, tSpec *ast.TypeSpec) []TextArea {
	switch specType := tSpec.Type.(type) {
	case *ast.StructType:
		return processStruct(p, dec, specType)
	}
	return nil
}

func processStruct(p Processor, dec *ast.GenDecl, structType *ast.StructType) (areas []TextArea) {
	if structType == nil {
		return nil
	}

	for _, field := range structType.Fields.List {
		var area = p.Field(field)
		if area != nil {
			areas = append(areas, area)
		}
	}

	var comments []*ast.Comment
	if dec.Doc != nil {
		comments = dec.Doc.List
	}
	var area = p.Struct(structType, comments)
	if area != nil {
		areas = append(areas, area)
	}
	return areas
}
