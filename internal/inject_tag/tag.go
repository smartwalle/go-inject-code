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
	iTagComment = regexp.MustCompile(`[\s\S^@]*@GoTag\(([^\)]+)\).*?`)
	rTagComment = regexp.MustCompile(`[\s\S^@]*@GoReTag\(([^\)]+)\).*?`)
	tagSplit    = regexp.MustCompile(`[\w_]+:"[^"]+"`)
	tagInject   = regexp.MustCompile("`.+`$")
)

// NewProcessField 生成字段的 tag 信息，包含两个功能：
// 1、根据字段的注释 @GoTag() 生成 tag，如：从 @GoTag(bson:"_id") 提取出 bson:"_id"；
// 2、根据字段的注释 @GoReTag() 替换 tag，如：从 @GoReTag(bson:"_id") 提取出 bson:"_id"，如果该字段有 bson tag，则替换该 bson tag 的内容为 _id，如果该字段没有 bson tag，则会添加 bson:"_id"；
// 3、根据参数 genTags 为字段生成 tag；
// 生成的 tag 不会覆盖原有的 tag，会追加在原有 tag 的后面，如果 tag 已经存在，则不会重复生成。
func NewProcessField(s string) internal.FieldProcessor {
	var nTags []string
	if s != "" {
		nTags = strings.Split(s, "|")
	}
	return func(field *ast.Field) internal.TextArea {
		var iTags = make([]string, 0, 2+len(nTags))
		var rTags = make([]string, 0, 2)

		// 从注释中提取要添加的 tag 信息
		if field.Doc != nil {
			for _, comment := range field.Doc.List {
				iTags, rTags = ParseTag(comment.Text, iTags, rTags)
			}
		}
		if field.Comment != nil {
			for _, comment := range field.Comment.List {
				iTags, rTags = ParseTag(comment.Text, iTags, rTags)
			}
		}

		if len(field.Names) > 0 {
			if field.Names[0].IsExported() {
				// 如果字段为可导出的（外部可访问），则为其自动生成指定的 tag 信息
				var name = internal.SnakeCase(field.Names[0].Name)
				for _, tag := range nTags {
					iTags = append(iTags, fmt.Sprintf("%s:\"%s\"", tag, name))
				}
			}
		}

		if len(iTags) == 0 && len(rTags) == 0 {
			return nil
		}

		// 获取字段原有的 tag 信息
		var mTag string
		if field.Tag != nil && len(field.Tag.Value) > 0 {
			mTag = field.Tag.Value[1 : len(field.Tag.Value)-1]
		}

		var nArea = &TextArea{
			start: int(field.Pos()) - 1,
			end:   int(field.End()) - 1,
			mTag:  mTag,
			iTag:  strings.Join(iTags, " "),
			rTag:  strings.Join(rTags, " "),
		}
		return nArea
	}
}

func ParseTag(text string, iTags, rTags []string) ([]string, []string) {
	if text == "" {
		return iTags, rTags
	}
	var ts = strings.Split(text, "@")

	for _, s := range ts {
		if s != "" {
			s = "@" + s
			var tag = FindTagString(s)
			if tag != "" {
				iTags = append(iTags, tag)
			}

			tag = FindReTagString(s)
			if tag != "" {
				rTags = append(rTags, tag)
			}
		}
	}
	return iTags, rTags
}

// FindTagString 从字符串中提取出要注入的 tag 字符串内容。
// 如：从 @GoTag(bson:"_id") 提取出 bson:"_id"。
func FindTagString(s string) (tag string) {
	var match = iTagComment.FindStringSubmatch(s)
	if len(match) == 2 {
		tag = match[1]
	}
	return
}

// FindReTagString 从字符串中提取出要替换的 tag 字符串内容。
// 如：从 @GoReTag(bson:"_id") 提取出 bson:"_id"。
func FindReTagString(s string) (tag string) {
	var match = rTagComment.FindStringSubmatch(s)
	if len(match) == 2 {
		tag = match[1]
	}
	return
}

type TextArea struct {
	start int
	end   int
	mTag  string // 原有 tag
	iTag  string // 新增 tag，从 @GoTag() 提取和参数 --tag 生成
	rTag  string // 替换 tag，从 @GoReTag() 提取
}

func (this *TextArea) Inject(content []byte) []byte {
	var iTags = NewTags(this.iTag)
	var rTags = NewTags(this.rTag)
	if len(iTags) == 0 && len(rTags) == 0 {
		return content
	}

	// 将字段原有的 tag 和要添加的 tag 进行合并
	var mTags = NewTags(this.mTag)
	var nTags = mTags.Merge(iTags, rTags)

	var text = make([]byte, this.end-this.start)
	copy(text, content[this.start:this.end])

	if this.mTag == "" {
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
	injected = append(injected, content[:this.start]...)
	injected = append(injected, text...)
	injected = append(injected, content[this.end:]...)
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

func NewTags(s string) Tags {
	var tags = tagSplit.FindAllString(s, -1)
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
