// 数据库表，当前简化设计使用一张表，涉及到的1:N的信息都通过json化方式存储
package schema

import (
	"time"
)

type Page struct {
	ID        uint64    `xorm:"bigint pk autoincr 'id'"`
	URL       string    `xorm:"varchar(2048) notnull unique(uk_url) 'url'"`
	Domain    string    `xorm:"varchar(256) notnull 'domain'"`
	State     uint8     `xorm:"int 'state'"`
	Remark    string    `xorm:"text 'remark'"`
	Paths     string    `xorm:"text 'paths'"`
	SubURLs   string    `xorm:"text 'sub_urls'"`
	FetchedAt time.Time `xorm:"datetime 'fetched_at'"`
	CreatedAt time.Time `xorm:"created notnull 'created_at'"`
	UpdatedAt time.Time `xorm:"updated notnull 'updated_at'"`
}

func (p *Page) TableName() string {
	return "pages"
}
