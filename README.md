# gin-pagination

#### 介绍
基于gorm实现的gin分页中间件


#### 使用说明

1.引入
```go
import "github.com/Ostaer/gin-pagination"
```

2.使用中间件
```go
app := gin.New()
paginationHandler := pagination.New(&pagination.CustomeParser{})
app.Get("/test", paginationHandler)
```

3.gorm引用查询数据
 
测试请求如[`middleware_test.go` 110-115行]

`
GET /test3?page=1&&size=3&&order=desc&&order_by=id&&is_child=true&&child_relation=1&&search.1.key=age&&search.1.value=5&&search.1.exactly=true&&search.2.key=email&&search.2.value=user-1,user-2&&search.2.exactly=false
`


查询处理如[`middleware_test.go` 211-214行]

```go
// SELECT count(*) FROM `users`  WHERE ((`age` = "5") or ((`email` like "%user-1%") or (`email` like "%user-2%")))
DB.Model(&User{}).Scopes(pagination.PaginationScope(query, ip.Count)).Count(&count)
// SELECT * FROM `users`  WHERE ((`age` = "5") or ((`email` like "%user-1%") or (`email` like "%user-2%"))) ORDER BY id desc LIMIT 3 OFFSET 0
DB.Model(&User{}).Scopes(pagination.PaginationScope(query)).Find(&users)
```

拼接query
```go
var query = pagination.PaginationQuery{
    PageNum: 1, PageSize: 10,
}
query.Search = append(query.Search, pagination.SearchGroup{
    SearchTerm: pagination.SearchTerm{
        Raw: &pagination.RawQuery{
            Query: "user_name = ? or nick_name = ?",
            Args: []interface{}{
                "张三", "四",
            },
        },
    },
})
```
自己组装gorm的Scope
```go
var scope = func(db *gorm.DB) *gorm.DB {
	d := &pagination.PaginationQueryDB{DB: db, Query: query}
	d.ParseLimit().ParsePagination().ParseOrder()
	return d.DB
}
a.conn.Model(&model.Account{}).Scopes(scope).Where("id in ?", ids).Find(&accounts)
```


更多例子
参考middleware_test.go文件

#### 文件说明

|文件|说明|
|--|--|
| model.go | 分页查询相关struct的定义, 如`PaginationQuery` |
| customParser.go | 负责解析查询参数到`PaginationQuery` |
| resolver.go | 负责解析`PaginationQuery`到gorm.DB Scopes |
| middleware.go | 实现兼容gin的中间件 |

