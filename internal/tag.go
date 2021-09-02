package internal

import (
	"bytes"
	"fmt"
	"strings"
)

type TagItem struct {
	key   string
	value string
}

type TagItems []TagItem

func (this TagItems) String() string {
	var tags = make([]string, 0, len(this))
	for _, item := range this {
		tags = append(tags, fmt.Sprintf(`%s:%s`, item.key, item.value))
	}
	return strings.Join(tags, " ")
}

func (this TagItems) Merge(tags TagItems) TagItems {
	var nTags = make([]TagItem, 0, len(this)+len(tags))
	for i := range this {
		var found = -1
		for j := range tags {
			if this[i].key == tags[j].key {
				found = j
				break
			}
		}

		if found == -1 {
			nTags = append(nTags, this[i])
		} else {
			nTags = append(nTags, tags[found])
			tags = append(tags[:found], tags[found+1:]...)
		}
	}
	return append(nTags, tags...)
}

func ParseTag(str string) TagItems {
	var tags = tagSplit.FindAllString(str, -1)
	var nTags = make([]TagItem, 0, 1)
	for _, tag := range tags {
		var pos = strings.Index(tag, ":")
		var item = TagItem{
			key:   tag[:pos],
			value: tag[pos+1:],
		}
		nTags = append(nTags, item)
	}
	return nTags
}

// InjectTag 注入 tag。
// 根据 area 中的位置信息，在 content 中找到相应的内容并进行替换。
func InjectTag(content []byte, area TextArea) (injected []byte) {
	var text = make([]byte, area.End-area.Start)
	copy(text, content[area.Start-1:area.End-1])

	cti := ParseTag(area.CurrentTag)
	iti := ParseTag(area.InjectTag)

	var nTags = cti.Merge(iti)

	if area.CurrentTag == "" {
		var buf = bytes.NewBuffer(text)
		buf.WriteString(" `")
		buf.WriteString(nTags.String())
		buf.WriteString("`")
		text = buf.Bytes()
	} else {
		text = tagInject.ReplaceAll(text, []byte(fmt.Sprintf("`%s`", nTags.String())))
	}

	injected = append(injected, content[:area.Start-1]...)
	injected = append(injected, text...)
	injected = append(injected, content[area.End-1:]...)
	return
}
