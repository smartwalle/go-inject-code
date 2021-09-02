package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"regexp"
)

var (
	tagComment = regexp.MustCompile(`^//.*?@GoTag\((.*)\)*`)
	tagSplit   = regexp.MustCompile(`[\w_]+:"[^"]+"`)
	tagInject  = regexp.MustCompile("`.+`$")
)

type TextArea struct {
	Start      int
	End        int
	CurrentTag string
	InjectTag  string
}

// TagFromComment 从字符串中提取出要注入的 tag 字符串内容。
// 如：从 @GoTag(bson:"_id") 提取出 bson:"_id"。
func TagFromComment(comment string) (tag string) {
	match := tagComment.FindStringSubmatch(comment)
	if len(match) == 2 {
		tag = match[1]
	}
	return
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
			var comments = make([]*ast.Comment, 0, 1)

			if field.Doc != nil {
				comments = append(comments, field.Doc.List...)
			}

			if field.Comment != nil {
				comments = append(comments, field.Comment.List...)
			}

			for _, comment := range comments {
				var tag = TagFromComment(comment.Text)
				if tag == "" {
					continue
				}

				var currentTag string
				if field.Tag != nil && len(field.Tag.Value) > 0 {
					currentTag = field.Tag.Value
					currentTag = field.Tag.Value[1 : len(currentTag)-1]
				}

				var nArea = TextArea{
					Start:      int(field.Pos()),
					End:        int(field.End()),
					CurrentTag: currentTag,
					InjectTag:  tag,
				}
				areas = append(areas, nArea)
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
		content = InjectTag(content, area)
	}

	if err = ioutil.WriteFile(path, content, 0644); err != nil {
		return err
	}

	return
}
