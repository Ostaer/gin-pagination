package pagination

import (
	"fmt"
	"gorm.io/gorm"
	"strings"
)

//
// 使用部分
//

// PaginationScope 解析PaginationQuery到gorm.DB Scopes
// 示例：
//          Query := &PaginationQuery{}
// 查询例子:
//          var result []yourModel
//          DB.Scopes(PaginationScope(Query)).Find(&result)
// 查询总数示例:
//          var c int
//          DB.Scopes(PaginationScope(Query, Count)).Count(&c)
func PaginationScope(query *PaginationQuery, options ...Option) func(db *gorm.DB) *gorm.DB {
	if Count.in(options) {
		// 只返回查询总条数的组合条件
		return func(db *gorm.DB) *gorm.DB {
			d := &PaginationQueryDB{db, query}
			d.ParseQuery()
			return d.DB
		}
	}
	// 否则返回查询的所有条件
	return func(db *gorm.DB) *gorm.DB {
		d := &PaginationQueryDB{db, query}
		d.ParseQuery().ParseLimit().ParsePagination().ParseOrder()
		return d.DB
	}
}

//
// 查询条件解析部分
//

// PaginationQueryDB 包含gorm.DB的结构体
type PaginationQueryDB struct {
	DB    *gorm.DB
	Query *PaginationQuery
}

// ParseQuery 解析查询条件
func (d *PaginationQueryDB) ParseQuery() *PaginationQueryDB {
	if len(d.Query.Search) > 0 {
		for _, search := range d.Query.Search {
			var expression = ""
			var query interface{}
			var args []interface{}
			if !search.IsChildQuery {
				query, args = _subSearchExpress(search.SearchTerm)
			} else {
				var expressChildren = make([]string, 0)
				for _, childSearch := range search.ChildSearch {
					q, as := _subSearchExpress(childSearch)
					if _, ok := q.(string); !ok {
						continue
					}
					expressChildren = append(expressChildren, q.(string))
					args = append(args, as...)
				}
				if search.ChildRelation == ChildSearchOr {
					expression = strings.Join(expressChildren, " or ")
				} else if search.ChildRelation == ChildSearchAnd {
					expression = strings.Join(expressChildren, " and ")
				}
				query = expression
			}
			d.DB = d.DB.Where(query, args...)
		}
	}
	return d
}

// ParseLimit 解析Limit
func (d *PaginationQueryDB) ParseLimit() *PaginationQueryDB {
	if d.Query.Limit > 0 {
		d.DB = d.DB.Limit(d.Query.Limit)
	}
	return d
}

// ParsePagination 解析分页条件
func (d *PaginationQueryDB) ParsePagination() *PaginationQueryDB {
	if d.Query.PageSize > 0 {
		d.DB = d.DB.Limit(d.Query.PageSize)
		if d.Query.PageNum > 0 {
			d.DB = d.DB.Offset((d.Query.PageNum - 1) * d.Query.PageSize)
		}
	}
	return d
}

// ParseOrder 解析排序
func (d *PaginationQueryDB) ParseOrder() *PaginationQueryDB {
	if len(d.Query.OrderBy) > 0 {
		if len(d.Query.Scope) == 0 {
			if strings.ToLower(d.Query.Order) == "asc" {
				d.DB = d.DB.Order(fmt.Sprintf("%s asc", d.Query.OrderBy))
			} else {
				d.DB = d.DB.Order(fmt.Sprintf("%s desc", d.Query.OrderBy))
			}
		} else {
			if strings.ToLower(d.Query.Order) == "asc" {
				d.DB = d.DB.Order(fmt.Sprintf("%s.%s asc", d.Query.Scope, d.Query.OrderBy))
			} else {
				d.DB = d.DB.Order(fmt.Sprintf("%s.%s desc", d.Query.Scope, d.Query.OrderBy))
			}
		}
	}
	return d
}

func _subSearchExpress(term SearchTerm) (query interface{}, args []interface{}) {
	if term.Raw != nil {
		return term.Raw.Query, term.Raw.Args
	} else {
		var sql string = ""
		if term.Comparision == "" {
			term.Comparision = "eq"
		}
		comparisionSymbol, ok := ComparisionMap[term.Comparision]
		if !ok {
			comparisionSymbol = "="
		}
		// 逗号分隔，or关系连接
		values := strings.Split(term.Value, ",")
		vNum := len(values)
		if len(term.Scope) > 0 {
			if term.Exactly {
				if comparisionSymbol == "in" || comparisionSymbol == "not in" {
					sql += "("
					sql += fmt.Sprintf("`%s`.`%s` %s ?", term.Scope, term.Key, comparisionSymbol)
					sql += ")"
					args = append(args, values)
				} else {
					for index, value := range values {
						if index == 0 {
							sql += "("
						}
						sql += fmt.Sprintf("(`%s`.`%s` %s ?)", term.Scope, term.Key, comparisionSymbol)
						args = append(args, value)
						if index < vNum-1 {
							sql += " or "
						}
						if index == vNum-1 {
							sql += ")"
						}
					}
				}
			} else {
				comparisionSymbol = "like"
				if term.Comparision == "ne" {
					comparisionSymbol = "not like"
				}
				for index, value := range values {
					if index == 0 {
						sql += "("
					}
					sql += fmt.Sprintf("(`%s`.`%s` %s ?)", term.Scope, term.Key, comparisionSymbol)
					args = append(args, "%"+value+"%")
					if index < vNum-1 {
						sql += " or "
					}
					if index == vNum-1 {
						sql += ")"
					}
				}
			}
		} else {
			if term.Exactly {
				if comparisionSymbol == "in" || comparisionSymbol == "not in" {
					sql += "("
					sql += fmt.Sprintf("`%s` %s ?", term.Key, comparisionSymbol)
					args = append(args, values)
					sql += ")"
				} else {
					for index, value := range values {
						if index == 0 {
							sql += "("
						}
						sql += fmt.Sprintf("(`%s` %s ?)", term.Key, comparisionSymbol)
						args = append(args, value)
						if index < vNum-1 {
							sql += " or "
						}
						if index == vNum-1 {
							sql += ")"
						}
					}
				}
			} else {
				comparisionSymbol = "like"
				if term.Comparision == "ne" {
					comparisionSymbol = "not like"
				}
				for index, value := range values {
					if index == 0 {
						sql += "("
					}
					sql += fmt.Sprintf("(`%s` %s ?)", term.Key, comparisionSymbol)
					args = append(args, "%"+value+"%")
					if index < vNum-1 {
						sql += " or "
					}
					if index == vNum-1 {
						sql += ")"
					}
				}
			}
		}
		return sql, args
	}
}
