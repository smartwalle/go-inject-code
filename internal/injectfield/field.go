package injectfield

import (
	"bytes"
	"github.com/smartwalle/go-inject-code/internal"
	"go/ast"
	"regexp"
)

var (
	fieldComment = regexp.MustCompile(`[\s\S^@]*@GoField\(\s*(\S+)\s+(.*)\s*\).*?`)
)

// FieldGenerator 生成字段信息
//
// 根据注释 @GoField() 生成字段，如：从 @GoField(Age int) 提取出 Age int
type FieldGenerator struct {
}

func NewFieldGenerator() *FieldGenerator {
	return &FieldGenerator{}
}

func (this *FieldGenerator) File(file *ast.File) internal.TextArea {
	return nil
}

func (this *FieldGenerator) Struct(structType *ast.StructType, comments []*ast.Comment) internal.TextArea {
	var exists = make(map[string]struct{}) // 用于记录结构体已有的字段，避免重复添加

	// 记录结构体已有字段
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 {
			exists[field.Names[0].Name] = struct{}{}
		}
	}

	var fields = make([]*Field, 0, len(comments))
	// 从注释中提取要添加的字段信息
	for _, comment := range comments {
		var field = findFieldString(comment.Text)
		if field == nil {
			continue
		}

		// 检测是否已经存在
		if _, ok := exists[field.Name]; ok {
			continue
		}

		exists[field.Name] = struct{}{}

		// 记录要添加的字段信息
		fields = append(fields, field)
	}

	if len(fields) == 0 {
		return nil
	}

	var nArea = &TextArea{}
	nArea.start = int(structType.Fields.Closing) - 1
	nArea.end = int(structType.Fields.Closing) - 1
	nArea.fields = fields
	return nArea
}

func (this *FieldGenerator) FieldList(fieldList *ast.FieldList) internal.TextArea {
	return nil
}

// findFieldString 从字符串中提取出要注入的字段内容。
//
// 如：从 @GoField(Age int) 提取出 Age int。
func findFieldString(s string) (field *Field) {
	var match = fieldComment.FindStringSubmatch(s)
	if len(match) == 3 {
		field = &Field{}
		field.Name = match[1]
		field.Type = match[2]
		return field
	}
	return nil
}

type TextArea struct {
	start  int
	end    int
	fields []*Field
}

func (this *TextArea) Inject(content []byte) []byte {
	if len(this.fields) == 0 {
		return content
	}

	var text = make([]byte, 0, 1024)
	var buf = bytes.NewBuffer(text)
	buf.WriteString("\t// inject fields \n")
	for _, field := range this.fields {
		buf.WriteByte('\t')
		buf.WriteString(field.Name)
		buf.WriteByte(' ')
		buf.WriteString(field.Type)
		buf.WriteByte('\n')
	}
	text = buf.Bytes()

	var injected = make([]byte, 0, len(content))
	injected = append(injected, content[:this.start]...)
	injected = append(injected, text...)
	injected = append(injected, content[this.end:]...)
	return injected
}

type Field struct {
	Name string
	Type string
}
