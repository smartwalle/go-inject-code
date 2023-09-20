package inject_import

import (
	"bytes"
	"github.com/smartwalle/go-inject-code/internal"
	"go/ast"
	"regexp"
)

var (
	importComment = regexp.MustCompile(`[\s\S^@]*@GoImport\((.*)\).*?`)
)

// NewImportGenerator 生成包导入信息
//
// 根据注释 @GoImport() 生成 import，如：从 @GoImport("time") 提取出 "time"
func NewImportGenerator() internal.ImportGenerator {
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

		var nImports = make([]string, 0, 4) // 用于记录要添加的包信息

		if f.Doc != nil {
			for _, comment := range f.Doc.List {
				nImports = ParseImport(exists, comment.Text, nImports)
			}
		}

		for _, group := range f.Comments {
			for _, comment := range group.List {
				nImports = ParseImport(exists, comment.Text, nImports)
			}
		}

		var nArea = &TextArea{}
		nArea.start = start
		nArea.nImport = nImports
		return nArea
	}
}

func ParseImport(exists map[string]struct{}, text string, nImports []string) []string {
	var nImport = FindImportString(text)
	if nImport == "" {
		return nImports
	}

	if _, ok := exists[nImport]; ok {
		return nImports
	}

	exists[nImport] = struct{}{}

	nImports = append(nImports, nImport)

	return nImports
}

// FindImportString 从字符串中提取出要导入的包内容。
//
// 如：从 @GoImport("time") 提取出 "time"。
func FindImportString(s string) string {
	var match = importComment.FindStringSubmatch(s)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

type TextArea struct {
	start   int
	nImport []string
}

func (this *TextArea) Inject(content []byte) []byte {
	if len(this.nImport) == 0 {
		return content
	}

	var text = make([]byte, 0, 1024)
	var buf = bytes.NewBuffer(text)
	buf.WriteString("\n// inject import \n")
	buf.WriteString("import (\n")
	for _, im := range this.nImport {
		buf.WriteByte('\t')
		buf.WriteString(im)
		buf.WriteByte('\n')
	}
	buf.WriteString(")\n")
	text = buf.Bytes()

	var injected = make([]byte, 0, len(content))
	injected = append(injected, content[:this.start]...)
	injected = append(injected, text...)
	injected = append(injected, content[this.start:]...)
	return injected
}
