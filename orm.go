package orm

import "fmt"

type Connect struct {
	Driver   string `json:"driver"`
	//
	Host     string `json:"host"`
	//端口
	Port     int `json:"port"`
	//用户名
	User     string `json:"user"`
	//密码
	Password string `json:"password"`
	//数据库名name
	Name     string `json:"name"`
}

func (p *Connect) String() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s~%s", p.User, p.Password, p.Host, p.Port, p.Name, p.Driver)
}

func (p *Connect) SqlString() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", p.User, p.Password, p.Host, p.Port, p.Name)
}

type Config map[string]Connect

type Page struct {
	Total     int64 `json:"total,omitempty"`
	PageTotal int   `json:"page_total,omitempty"`
	Page      int   `json:"page,omitempty"`
	PageSize  int   `json:"page_size,omitempty"`
}

type Orm struct {
	config Config
}

// 指定模型的入口
func (p *Orm) Model(mo interface{}) *Model {
	return p.newModel().Model(mo)
}

// 也可以不指定模型,但必须指定Table
func (p *Orm) Table(table string) *Model {
	return p.newModel().Table(table)
}

func (p *Orm) newModel() *Model {
	m := &Model{
		config: p.config,
	}
	return m
}

func New(config Config) *Orm {
	o := Orm{
		config:config,
	}
	return &o
}