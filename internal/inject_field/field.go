package inject_field

import (
	"bytes"
	"github.com/smartwalle/go-inject-code/internal"
	"go/ast"
	"regexp"
)

var (
	fieldComment = regexp.MustCompile(`^//.*?@GoField\(\s*(\S+)\s+(\S+)\s*\).*?`)
)

// NewProcessStruct 生成字段信息
func NewProcessStruct() internal.StructProcessor {
	return func(s *ast.StructType, comments []*ast.Comment) internal.TextArea {
		var exists = make(map[string]struct{}) // 用于记录结构体已有的字段，避免重复添加

		// 记录结构体已有字段
		for _, field := range s.Fields.List {
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
		nArea.Start = int(s.Fields.Closing) - 1
		nArea.End = int(s.Fields.Closing) - 1
		nArea.InjectField = fields
		return nArea
	}
}

// findFieldString 从字符串中提取出要注入的字段内容。
// 如：从 @GoField(Age int) 提取出 Age int。
func findFieldString(comment string) (field *Field) {
	var match = fieldComment.FindStringSubmatch(comment)
	if len(match) == 3 {
		field = &Field{}
		field.Name = match[1]
		field.Type = match[2]
		return field
	}
	return nil
}

type TextArea struct {
	Start       int
	End         int
	InjectField []*Field
}

func (this *TextArea) Inject(content []byte) []byte {
	if len(this.InjectField) == 0 {
		return content
	}

	var text = make([]byte, 0, 1024)
	var buf = bytes.NewBuffer(text)
	buf.WriteString("\t// inject fields \n")
	for _, field := range this.InjectField {
		buf.WriteByte('\t')
		buf.WriteString(field.Name)
		buf.WriteByte(' ')
		buf.WriteString(field.Type)
		buf.WriteByte('\n')
	}
	text = buf.Bytes()

	var injected = make([]byte, 0, len(content))
	injected = append(injected, content[:this.Start]...)
	injected = append(injected, text...)
	injected = append(injected, content[this.End:]...)

	return injected
}

type Field struct {
	Name string
	Type string
}
