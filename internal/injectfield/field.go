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

// BuildFieldProcessor 生成字段信息
//
// 根据注释 @GoField() 生成字段，如：从 @GoField(Age int) 提取出 Age int
type BuildFieldProcessor struct {
}

func NewBuildFieldProcessor() *BuildFieldProcessor {
	return &BuildFieldProcessor{}
}

func (p *BuildFieldProcessor) File(file *ast.File) internal.TextArea {
	return nil
}

func (p *BuildFieldProcessor) Struct(structType *ast.StructType, comments []*ast.Comment) internal.TextArea {
	var fields = make(map[string]struct{}) // 用于记录结构体已有的字段，避免重复添加

	// 记录结构体已有字段
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 {
			fields[field.Names[0].Name] = struct{}{}
		}
	}

	var nFields = make([]*Field, 0, len(comments))
	// 从注释中提取要添加的字段信息
	for _, comment := range comments {
		var field = findField(comment.Text)
		if field == nil {
			continue
		}

		// 检测是否已经存在
		if _, ok := fields[field.Name]; ok {
			continue
		}

		fields[field.Name] = struct{}{}

		// 记录要添加的字段信息
		nFields = append(nFields, field)
	}

	if len(nFields) == 0 {
		return nil
	}

	var nArea = &TextArea{}
	nArea.start = int(structType.Fields.Closing) - 1
	nArea.end = int(structType.Fields.Closing) - 1
	nArea.fields = nFields
	return nArea
}

func (p *BuildFieldProcessor) FieldList(fieldList *ast.FieldList) internal.TextArea {
	return nil
}

// findField 从字符串中提取出要注入的字段内容。
//
// 如：从 @GoField(Age int) 提取出 Age int。
func findField(s string) (field *Field) {
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

func (area *TextArea) Inject(content []byte) []byte {
	if len(area.fields) == 0 {
		return content
	}

	var text = make([]byte, 0, len(content)+1024)
	var buf = bytes.NewBuffer(text)

	buf.Write(content[:area.start])

	buf.WriteString("\t// inject fields \n")
	for _, field := range area.fields {
		buf.WriteByte('\t')
		buf.WriteString(field.Name)
		buf.WriteByte(' ')
		buf.WriteString(field.Type)
		buf.WriteByte('\n')
	}

	buf.Write(content[area.end:])
	return buf.Bytes()
}

type Field struct {
	Name string
	Type string
}
