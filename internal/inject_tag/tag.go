package inject_tag

import (
	"bytes"
	"fmt"
	"github.com/smartwalle/go-inject-code/internal"
	"go/ast"
	"regexp"
	"strings"
)

var (
	tagComment   = regexp.MustCompile(`[\s\S^@]*@GoTag\(([^\)]+)\).*?`)
	reTagComment = regexp.MustCompile(`[\s\S^@]*@GoReTag\(([^\)]+)\).*?`)
	tagSplit     = regexp.MustCompile(`[\w_]+:"[^"]+"`)
	tagInject    = regexp.MustCompile("`.+`$")
)

// NewProcessField 生成字段的 tag 信息，包含两个功能：
// 1、根据字段的注释 @GoTag() 生成 tag，如：从 @GoTag(bson:"_id") 提取出 bson:"_id"；
// 2、根据字段的注释 @GoReTag() 替换 tag，如：从 @GoReTag(bson:"_id") 提取出 bson:"_id"，如果该字段有 bson tag，则替换该 bson tag 的内容为 _id，如果该字段没有 bson tag，则会添加 bson:"_id"；
// 3、根据参数 genTags 为字段生成 tag；
// 生成的 tag 不会覆盖原有的 tag，会追加在原有 tag 的后面，如果 tag 已经存在，则不会重复生成。
func NewProcessField(genTags []string) internal.FieldProcessor {
	return func(field *ast.Field) internal.TextArea {
		var tags = make([]string, 0, 2+len(genTags))
		var rTags = make([]string, 0, 2)

		// 从注释中提取要添加的 tag 信息
		if field.Doc != nil {
			for _, comment := range field.Doc.List {
				tags, rTags = SplitTag(comment.Text, tags, rTags)
			}
		}
		if field.Comment != nil {
			for _, comment := range field.Comment.List {
				tags, rTags = SplitTag(comment.Text, tags, rTags)
			}
		}

		if len(field.Names) > 0 {
			if field.Names[0].IsExported() {
				// 如果字段为可导出的（外部可访问），则为其自动生成指定的 tag 信息
				var name = internal.SnakeCase(field.Names[0].Name)
				for _, tag := range genTags {
					tags = append(tags, fmt.Sprintf("%s:\"%s\"", tag, name))
				}
			}
		}

		if len(tags) == 0 && len(rTags) == 0 {
			return nil
		}

		// 获取字段原有的 tag 信息
		var currentTag string
		if field.Tag != nil && len(field.Tag.Value) > 0 {
			currentTag = field.Tag.Value[1 : len(field.Tag.Value)-1]
		}

		var nArea = &TextArea{
			Start:      int(field.Pos()) - 1,
			End:        int(field.End()) - 1,
			CurrentTag: currentTag,
			InjectTag:  strings.Join(tags, " "),
			ReTag:      strings.Join(rTags, " "),
		}
		return nArea
	}
}

func SplitTag(text string, tags, rTags []string) ([]string, []string) {
	if text == "" {
		return tags, rTags
	}
	var ts = strings.Split(text, "@")

	for _, s := range ts {
		if s != "" {
			s = "@" + s
			var tag = FindTagString(s)
			if tag != "" {
				tags = append(tags, tag)
			}

			tag = FindReTagString(s)
			if tag != "" {
				rTags = append(rTags, tag)
			}
		}
	}
	return tags, rTags
}

// FindTagString 从字符串中提取出要注入的 tag 字符串内容。
// 如：从 @GoTag(bson:"_id") 提取出 bson:"_id"。
func FindTagString(comment string) (tag string) {
	var match = tagComment.FindStringSubmatch(comment)
	if len(match) == 2 {
		tag = match[1]
	}
	return
}

// FindReTagString 从字符串中提取出要替换的 tag 字符串内容。
// 如：从 @GoReTag(bson:"_id") 提取出 bson:"_id"。
func FindReTagString(comment string) (tag string) {
	var match = reTagComment.FindStringSubmatch(comment)
	if len(match) == 2 {
		tag = match[1]
	}
	return
}

type TextArea struct {
	Start      int
	End        int
	CurrentTag string
	InjectTag  string
	ReTag      string
}

func (this *TextArea) Inject(content []byte) []byte {
	var injectTags = parseTags(this.InjectTag)
	var reTags = parseTags(this.ReTag)
	if len(injectTags) == 0 && len(reTags) == 0 {
		return content
	}

	// 将字段原有的 tag 和要添加的 tag 进行合并
	var currentTags = parseTags(this.CurrentTag)
	var nTags = currentTags.Merge(injectTags, reTags)

	var text = make([]byte, this.End-this.Start)
	copy(text, content[this.Start:this.End])

	if this.CurrentTag == "" {
		// 如果字段原来没有任何 tag，则生成完整的 tag 信息
		var buf = bytes.NewBuffer(text)
		buf.WriteString(" `")
		buf.WriteString(nTags.String())
		buf.WriteString("`")
		text = buf.Bytes()
	} else {
		// 如果字段原来有 tag，则替换 tag 内容
		text = tagInject.ReplaceAll(text, []byte(fmt.Sprintf("`%s`", nTags.String())))
	}

	var injected = make([]byte, 0, len(content)+len(text))
	injected = append(injected, content[:this.Start]...)
	injected = append(injected, text...)
	injected = append(injected, content[this.End:]...)
	return injected
}

type Tag struct {
	key   string
	value string
}

type Tags []Tag

func (this Tags) String() string {
	var tags = make([]string, 0, len(this))
	for _, item := range this {
		tags = append(tags, fmt.Sprintf(`%s:%s`, item.key, item.value))
	}
	return strings.Join(tags, " ")
}

func (this Tags) Merge(tags, rTags Tags) Tags {
	var nTags = make([]Tag, 0, len(this)+len(tags))

	// 方便后续查找，转换成 map
	var replace = make(map[string]Tag)
	for _, t := range rTags {
		replace[t.key] = t
	}

	var exists = make(map[string]struct{})
	for _, tag := range this {
		exists[tag.key] = struct{}{}
		if rTag, ok := replace[tag.key]; ok {
			// 如果在需要替换的列表中，则使用替换列表中的内容
			nTags = append(nTags, rTag)
		} else {
			nTags = append(nTags, tag)
		}
		delete(replace, tag.key)
	}

	for _, tag := range tags {
		if _, ok := exists[tag.key]; ok == false {
			exists[tag.key] = struct{}{}
			if rTag, ok := replace[tag.key]; ok {
				nTags = append(nTags, rTag)
			} else {
				nTags = append(nTags, tag)
			}
			delete(replace, tag.key)
		}
	}

	for _, tag := range replace {
		if _, ok := exists[tag.key]; ok == false {
			exists[tag.key] = struct{}{}
			nTags = append(nTags, tag)
		}
	}
	return nTags
}

func parseTags(str string) Tags {
	var tags = tagSplit.FindAllString(str, -1)
	var nTags = make([]Tag, 0, 1)
	for _, tag := range tags {
		var pos = strings.Index(tag, ":")
		var item = Tag{
			key:   tag[:pos],
			value: tag[pos+1:],
		}
		nTags = append(nTags, item)
	}
	return nTags
}
