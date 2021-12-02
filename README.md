## go-inject-code

根据注释生成代码，目前可以生成 Tag、Field 和导入包。

本项目单独使用意义不大，主要配合 Protobuf 文件进行使用，比如为 Protobuf 定义的 message 类型添加新的 Tag 或者修改其已有 Tag。

#### 例子

```protobuf
message User {
  int64 Id = 1; // @GoTag(bson:"_id")
  string Name = 2; // @GoTag(bson:"name")
}
```

当使用 protoc 工具编译上述代码后，可得到以下类似 Golang 代码:

```go
type User struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    Id            int64               `protobuf:"varint,1,opt,name=Id,proto3" json:"Id,omitempty"` // @GoTag(bson:"_id")
    Name          string              `protobuf:"bytes,2,opt,name=Name,proto3" json:"Name,omitempty"` // @GoTag(bson:"name")
}
```

然后使用 go-inject-code 工具对该 Golang 代码进行处理，可得到以下类似 Golang 代码:

```go
type User struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    Id            int64               `protobuf:"varint,1,opt,name=Id,proto3" json:"Id,omitempty" bson:"_id"` // @GoTag(bson:"_id")
    Name          string              `protobuf:"bytes,2,opt,name=Name,proto3" json:"Name,omitempty" bson:"name"` // @GoTag(bson:"name")
}
```

为 protoc 生成的 Golang 代码添加了 bson Tag。

## 生成 Tag

### @GoTag

用于生成 Tag，支持生成多个 Tag。

如果该字段已有需要生成的 Tag，则不会重复生成。

##### 语法

```go
type User struct {
    Name string // @GoTag(bson:"name"  json:"name")
}
```

### @GoReTag

用于生成或者替换 Tag，支持生成或者替换多个 Tag。

和 @GoTag 相比，如果该字段已有需要生成的 Tag，会使用新的 Tag 替换原来的 Tag。

##### 语法

```go
type User struct {
    Name string `json:"name"` // @GoReTag(bson:"name"  json:"new_name")
}
```

## 生成 Field

### @GoField

用于为结构体(struct)生成新的字段。

如果该结构体已有需要生成的字段，则不会重复生成。

##### 语法
```go
// @GoField(Age int)
type User struct {
    Name string 
}
```

## 导入包

### @GoImport

用于导入新的包

##### 语法
```go
// @GoImport("time")
```

## 使用

### 安装

在终端中执行以下命令
```shell
go install github.com/smartwalle/go-inject-code@latest
```

确认在 GOPATH/bin 目录中存在 go-inject-code。

### 生成

进入需要处理 Golang 代码所在目录，执行以下命令
```shell
go-inject-code --input "./"
```

更多参数执行以下命令查看
```shell
go-inject-code --help 
```