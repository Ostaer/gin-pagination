package pagination

import (
	"fmt"
	"github.com/gavv/httpexpect/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

var DB *gorm.DB

func TestMiddlewareParser(t *testing.T) {
	app := gin.New()
	pagination := New(&CustomeParser{})
	app.GET("/test", pagination, testHandler)
	// 测试中间件 负责解析成 `PaginationQuery`
	server := httptest.NewServer(app)
	defer server.Close()

	e := httpexpect.New(t, server.URL)
	r := e.GET("/test").
		WithQueryString("page=1&&size=3&&order=desc&&order_by=id").
		WithQueryString("&&search.1.key=age&&search.1.value=5&&search.1.exactly=true").
		Expect().Status(http.StatusOK)
	exceptedJson := PaginationQuery{
		PageSize: 3,
		PageNum:  1,
		OrderBy:  "id",
		Order:    "desc",
		Limit:    10,
		Search: []SearchGroup{
			{
				SearchTerm: SearchTerm{
					Key:         "age",
					Value:       "5",
					Scope:       "",
					Comparision: "eq",
					Exactly:     true,
				},
				IsChildQuery:  false,
				ChildSearch:   nil,
				ChildRelation: 0,
			},
		},
		Scope: "",
	}
	r.JSON().Equal(exceptedJson)
}

func TestMiddlewareSimpleQuery(t *testing.T) {
	// 测试数据库查询方法
	db, err := OpenTestConnection()
	if err != nil {
		t.Error(err)
	}
	//DB.LogMode(true)

	_ = db.Migrator().DropTable(&User{})
	_ = db.Migrator().AutoMigrate(&User{})
	for i := 1; i <= 100; i++ {
		err := db.Create(&User{
			Name:  fmt.Sprintf("user-%d", i),
			Email: fmt.Sprintf("user-%d@email.com", i),
			Age:   i % 10,
		}).Error
		if err != nil {
			t.Error(err)
		}
	}

	app := gin.New()
	pagination := New(&CustomeParser{})
	app.GET("/test2", pagination, testHandler2)
	server := httptest.NewServer(app)
	e := httpexpect.New(t, server.URL)
	r := e.GET("/test2").
		WithQueryString("page=1&&size=3&&order=desc&&order_by=id").
		WithQueryString("&&search.1.key=age&&search.1.value=5&&search.1.exactly=true&&search.1.comparision=lt").
		WithQueryString("&&search.2.key=email&&search.2.value=user&&search.2.exactly=false").
		Expect().Status(http.StatusOK)
	exceptedJson := HandlerIDsResponse{Data: []uint{100, 94, 93, 50}}
	r.JSON().Equal(exceptedJson)
}

func TestMiddlewareChildQuery(t *testing.T) {
	// 测试数据库查询方法
	db, err := OpenTestConnection()
	if err != nil {
		t.Error(err)
	}
	//DB.LogMode(true)
	_ = db.Migrator().DropTable(&User{})
	_ = db.Migrator().AutoMigrate(&User{})
	for i := 1; i <= 100; i++ {
		db.Create(&User{
			Name:  fmt.Sprintf("user-%d", i),
			Email: fmt.Sprintf("user-%d@email.com", i),
			Age:   i % 10,
		})
	}

	app := gin.New()
	pagination := New(&CustomeParser{})
	app.GET("/test3", pagination, testHandler3)
	server := httptest.NewServer(app)
	e := httpexpect.New(t, server.URL)
	r := e.GET("/test3").
		WithQueryString("page=1&&size=3&&order=desc&&order_by=id").
		WithQueryString("is_child=true&&child_relation=1").
		WithQueryString("&&search.1.key=age&&search.1.value=5&&search.1.exactly=true").
		WithQueryString("&&search.2.key=email&&search.2.value=user-1&&search.2.exactly=false").
		Expect().Status(http.StatusOK)
	exceptedJson := HandlerIDsResponse{Data: []uint{100, 95, 85, 21}}
	r.JSON().Equal(exceptedJson)
}

func TestMiddlewareChildQueryMiddlewareJoinQuery(t *testing.T) {
	// 测试数据库查询方法
	db, err := OpenTestConnection()
	if err != nil {
		t.Error(err)
	}

	_ = db.Migrator().DropTable(&User{}, &Card{})
	_ = db.Migrator().AutoMigrate(&User{}, &Card{})
	for i := 1; i <= 100; i++ {
		db.Create(&User{
			Name:  fmt.Sprintf("user-%d", i),
			Email: fmt.Sprintf("user-%d@email.com", i),
			Age:   i % 10,
		})
		db.Create(&Card{
			UserID:  uint(i),
			CardNum: uint(100-i) % 10,
		})
	}

	app := gin.New()
	pagination := New(&CustomeParser{})
	app.GET("/test4", pagination, testHandler4)
	server := httptest.NewServer(app)
	e := httpexpect.New(t, server.URL)
	r := e.GET("/test4").
		WithQueryString("page=1&&size=3&&order=desc&&order_by=id&&scope=users").
		WithQueryString("is_child=true&&child_relation=1").
		WithQueryString("&&search.1.key=age&&search.1.value=5&&search.1.exactly=true&&search.1.scope=users").
		WithQueryString("&&search.2.key=card_num&&search.2.value=3&&search.2.exactly=true&&search.2.scope=cards").
		Expect().Status(http.StatusOK)
	exceptedJson := HandlerIDsResponse{Data: []uint{97, 95, 87, 20}}
	r.JSON().Equal(exceptedJson)
}

func TestRawQuery(t *testing.T) {
	// 测试数据库查询方法
	db, err := OpenTestConnection()
	if err != nil {
		t.Error(err)
	}
	//DB.LogMode(true)

	_ = db.Migrator().DropTable(&User{})
	_ = db.Migrator().AutoMigrate(&User{})
	for i := 1; i <= 100; i++ {
		err := db.Create(&User{
			Name:  fmt.Sprintf("user-%d", i),
			Email: fmt.Sprintf("user-%d@email.com", i),
			Age:   i,
		}).Error
		if err != nil {
			t.Error(err)
		}
	}
	query := &PaginationQuery{
		Search: []SearchGroup{SearchGroup{
			SearchTerm: SearchTerm{Raw: &RawQuery{
				Query: &User{Age: 5}, // `age` = 5
			}},
		}},
	}

	var users []User
	db.Model(&User{}).Scopes(PaginationScope(query)).Find(&users)
	if !(len(users) == 1 && users[0].Name == fmt.Sprintf("user-%d", 5)) {
		t.Error("raw Query error", users[0].Name)
	}
}

type User struct {
	ID    uint   `gorm:"primary_key" json:"-"`
	Name  string `gorm:"size:255;unique_index:name;column:name;" json:"name"`
	Email string `gorm:"size:32;column:email;" form:"email" json:"email"`
	Age   int
	Card  Card
}

type Card struct {
	ID      uint `gorm:"primary_key" json:"-"`
	UserID  uint
	CardNum uint
}

func testHandler(c *gin.Context) {
	paginationQuery, _ := c.Get(PaginationQueryContextKey)
	c.JSON(http.StatusOK, paginationQuery)
	c.Next()
}

type HandlerIDsResponse struct {
	Data []uint `json:"json"`
}

func testHandler2(c *gin.Context) {
	paginationQuery, _ := c.Get(PaginationQueryContextKey)
	query := paginationQuery.(*PaginationQuery)
	var (
		users []User
		count int64
	)
	// SELECT count(*) FROM `users`  WHERE ((`age` < "5")) AND ((`email` like "%user%"))
	DB.Model(&User{}).Scopes(PaginationScope(query, Count)).Count(&count)
	//SELECT * FROM `users`  WHERE ((`age` < "5")) AND ((`email` like "%user%")) ORDER BY id desc LIMIT 3 OFFSET 0
	DB.Model(&User{}).Scopes(PaginationScope(query)).Find(&users)

	var ids []uint
	for _, user := range users {
		ids = append(ids, user.ID) //[100, 94, 93]
	}
	ids = append(ids, uint(count)) // [100, 94, 93, 50]
	c.JSON(http.StatusOK, HandlerIDsResponse{
		Data: ids,
	})
	c.Next()
}

func testHandler3(c *gin.Context) {
	paginationQuery, _ := c.Get(PaginationQueryContextKey)
	query := paginationQuery.(*PaginationQuery)
	var (
		users []User
		count int64
	)
	// SELECT count(*) FROM `users`  WHERE ((`age` = "5") or (`email` like "%user-1%"))
	DB.Model(&User{}).Scopes(PaginationScope(query, Count)).Count(&count)
	// SELECT * FROM `users`  WHERE ((`age` = "5") or (`email` like "%user-1%")) ORDER BY id desc LIMIT 3 OFFSET 0
	DB.Model(&User{}).Scopes(PaginationScope(query)).Find(&users)

	var ids []uint
	for _, user := range users {
		ids = append(ids, user.ID) //[100, 95, 85]
	}
	ids = append(ids, uint(count)) // [100, 95, 85, 31]
	c.JSON(http.StatusOK, HandlerIDsResponse{
		Data: ids,
	})
	c.Next()
}

func testHandler4(c *gin.Context) {
	paginationQuery, _ := c.Get(PaginationQueryContextKey)
	query := paginationQuery.(*PaginationQuery)
	var (
		users []User
		count int64
	)
	// SELECT count(*) FROM `users` left join cards on cards.user_id = users.id WHERE ((`users`.`age` = "5") or (`cards`.`card_num` = "3"))
	DB.Model(&User{}).Joins("left join cards on cards.user_id = users.id").Scopes(PaginationScope(query, Count)).Count(&count)
	// SELECT `users`.* FROM `users` left join cards on cards.user_id = users.id WHERE ((`users`.`age` = "5") or (`cards`.`card_num` = "3")) ORDER BY users.id desc LIMIT 3 OFFSET 0
	DB.Model(&User{}).Joins("left join cards on cards.user_id = users.id").Scopes(PaginationScope(query)).Find(&users)

	var ids []uint
	for _, user := range users {
		ids = append(ids, user.ID) //[97, 95, 87]
	}
	ids = append(ids, uint(count)) // [97, 95, 87, 20]
	c.JSON(http.StatusOK, HandlerIDsResponse{
		Data: ids,
	})
	c.Next()
}

func OpenTestConnection() (db *gorm.DB, err error) {
	dbDSN := os.Getenv("DSN")
	switch os.Getenv("DIALECT") {
	case "mysql":
		if dbDSN == "" {
			dbDSN = "user:password@tcp(localhost:3306)/gin?charset=utf8&parseTime=True"
		}
		db, err = gorm.Open(mysql.Open(dbDSN), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
	default:
		db, err = gorm.Open(sqlite.Open(filepath.Join(os.TempDir(), "test.DB")), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
	}

	DB = db
	return
}
