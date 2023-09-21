package examples

// 打开终端进入本文件所在目录，然后执行 go-inject-code --input "./"

// @GoImport("time")

// User
// @GoField(Age int)
// @GoField(CreatedAt *time.Time)
type User struct {
	Name string `json:"name" bson:"name"` // @GoReTag(bson:"name"  json:"new_name")
}
