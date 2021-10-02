package inject_import

import (
	"bytes"
	"fmt"
	"github.com/smartwalle/go-inject-code/internal"
	"go/ast"
	"regexp"
)

var (
	importComment = regexp.MustCompile(`^//\s*@GoImport\((.*)\).*?`)
)

// NewProcessImport 生成包导入信息
func NewProcessImport() internal.ImportProcessor {
	return func(f *ast.File) internal.TextArea {
		var exists = make(map[string]struct{}) // 用于记录已导入的包，避免重复导入

		var start = 0 // 用于记录包导入的位置
		for _, im := range f.Imports {
			if im.Name != nil {
				exists[im.Name.Name+" "+im.Path.Value] = struct{}{}
			} else {
				exists[im.Path.Value] = struct{}{}
			}

			start = int(im.End()) + 1 // 如果原来有导入包，则追加在其后面
		}

		if start == 0 {
			// 如果原来没有导入包，则重包名后开始
			start = int(f.Name.End())
		}

		var imports = make([]string, 0, 4) // 用于记录要添加的包信息

		if f.Doc != nil {
			for _, comment := range f.Doc.List {
				imports = parseImportString(exists, comment.Text, imports)
			}
		}

		for _, group := range f.Comments {
			for _, comment := range group.List {
				imports = parseImportString(exists, comment.Text, imports)
			}
		}

		fmt.Println(imports)

		var nArea = &TextArea{}
		nArea.Start = start
		nArea.InjectImport = imports
		return nArea
	}
}

func parseImportString(exists map[string]struct{}, comment string, imports []string) []string {
	var in = findImportString(comment)
	if in == "" {
		return imports
	}

	if _, ok := exists[in]; ok {
		return imports
	}

	exists[in] = struct{}{}

	imports = append(imports, in)

	return imports
}

// findImportString 从字符串中提取出要导入的包内容。
// 如：从 @GoImport("time") 提取出 "time"。
func findImportString(comment string) string {
	var match = importComment.FindStringSubmatch(comment)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

type TextArea struct {
	Start        int
	InjectImport []string
}

func (this *TextArea) Inject(content []byte) []byte {
	if len(this.InjectImport) == 0 {
		return content
	}

	var text = make([]byte, 0, 1024)
	var buf = bytes.NewBuffer(text)
	buf.WriteString("\n// inject import \n")
	buf.WriteString("import (\n")
	for _, im := range this.InjectImport {
		buf.WriteByte('\t')
		buf.WriteString(im)
		buf.WriteByte('\n')
	}
	buf.WriteString(")")
	text = buf.Bytes()

	var injected = make([]byte, 0, len(content))
	injected = append(injected, content[:this.Start]...)
	injected = append(injected, text...)
	injected = append(injected, content[this.Start:]...)
	return injected
}
