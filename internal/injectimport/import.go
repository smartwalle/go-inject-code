package injectimport

import (
	"bytes"
	"github.com/smartwalle/go-inject-code/internal"
	"go/ast"
	"regexp"
)

var (
	importComment = regexp.MustCompile(`[\s\S^@]*@GoImport\((.*)\).*?`)
)

// BuildImportProcessor 生成包导入信息
//
// 根据注释 @GoImport() 生成 import，如：从 @GoImport("time") 提取出 "time"
type BuildImportProcessor struct {
}

func NewBuildImportProcessor() *BuildImportProcessor {
	return &BuildImportProcessor{}
}

func (p *BuildImportProcessor) File(file *ast.File) internal.TextArea {
	var imports = make(map[string]struct{}) // 用于记录已导入的包，避免重复导入

	var start = 0 // 用于记录包导入的位置
	for _, im := range file.Imports {
		if im.Name != nil {
			imports[im.Name.Name+" "+im.Path.Value] = struct{}{}
		} else {
			imports[im.Path.Value] = struct{}{}
		}

		start = int(im.End()) + 1 // 如果原来有导入包，则追加在其后面
	}

	if start == 0 {
		// 如果原来没有导入包，则从包名后开始
		start = int(file.Name.End())
	}

	var nImports = make([]string, 0, 4) // 用于记录要添加的包信息

	if file.Doc != nil {
		for _, comment := range file.Doc.List {
			nImports = parseImport(imports, comment.Text, nImports)
		}
	}

	for _, group := range file.Comments {
		for _, comment := range group.List {
			nImports = parseImport(imports, comment.Text, nImports)
		}
	}

	var nArea = &TextArea{}
	nArea.start = start
	nArea.nImport = nImports
	return nArea
}

func (p *BuildImportProcessor) Struct(structType *ast.StructType, comments []*ast.Comment) internal.TextArea {
	return nil
}

func (p *BuildImportProcessor) FieldList(fieldList *ast.FieldList) internal.TextArea {
	return nil
}

func parseImport(imports map[string]struct{}, text string, nImports []string) []string {
	var nImport = findImport(text)
	if nImport == "" {
		return nImports
	}

	if _, ok := imports[nImport]; ok {
		return nImports
	}

	imports[nImport] = struct{}{}

	nImports = append(nImports, nImport)

	return nImports
}

// findImport 从字符串中提取出要导入的包内容。
//
// 如：从 @GoImport("time") 提取出 "time"。
func findImport(s string) string {
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

func (area *TextArea) Inject(content []byte) []byte {
	if len(area.nImport) == 0 {
		return content
	}

	var text = make([]byte, 0, len(content)+1024)
	var buf = bytes.NewBuffer(text)

	buf.Write(content[:area.start])

	buf.WriteString("\n// inject import \n")
	buf.WriteString("import (\n")
	for _, im := range area.nImport {
		buf.WriteByte('\t')
		buf.WriteString(im)
		buf.WriteByte('\n')
	}
	buf.WriteString(")\n")

	buf.Write(content[area.start:])
	return buf.Bytes()
}
