// NOTE: 注意此处不设置interface
// 当前controller等任务与dbstorage紧耦合，部分逻辑处理必须糅合在一起（主要是事务性考量）
// 当前的事务性原则为： 保留上一次操作记录，仅修改paths部分，其余不做改动
// 未来优化时必须根据db存储的选型以及产品思路来修改此部分逻辑
// NOTE: 没有连接池等处理
package dbstorage

import (
	"errors"
	"time"

	"github.com/go-xorm/xorm"
	_ "github.com/lib/pq" // pg driver

	"github.com/andrewyi/crawler/src/dbstorage/schema"
	"github.com/andrewyi/crawler/src/enum"
)

var (
	ErrDataNotExist = errors.New("data not exist")
	ErrDataExist    = errors.New("data exist")
)

type SimpleDBStorage struct {
	engine *xorm.Engine
}

// dbURL sample: postgres://postgres:root@localhost:5432/testdb?sslmode=disable
func NewSimpleDBStorage(dbURL string) (*SimpleDBStorage, error) {
	var s = &SimpleDBStorage{}
	engine, err := xorm.NewEngine("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	s.engine = engine
	s.engine.SetMaxIdleConns(3)
	// d.engine.ShowSQL(true) // debug
	return &SimpleDBStorage{}, nil
}

func (s *SimpleDBStorage) Close() error {
	return s.engine.Close()
}

// Transaction 数据库事务
type Transaction struct {
	sess *xorm.Session
}

// NewTransaction open a transaction with engine
// all dao functions should use transaction to crud
func (s *SimpleDBStorage) NewTransaction() (*Transaction, error) {
	t := &Transaction{
		sess: s.engine.NewSession(),
	}
	err := t.sess.Begin()
	return t, err
}

func (t *Transaction) Commit() error {
	return t.sess.Commit()
}

func (t *Transaction) Rollback() error {
	return t.sess.Rollback()
}

func (t *Transaction) Close() {
	_ = t.sess.Rollback()
	t.sess.Close()
}

func (t *Transaction) GetPageWithLock(url string) (*schema.Page, error) {

	pages := make([]*schema.Page, 0)
	if err := t.sess.SQL(
		"select * from pages where url = ? for update skip locked",
		url).Find(&pages); err != nil {

		return nil, err
	}

	if len(pages) == 0 {
		return nil, ErrDataNotExist
	}

	return pages[0], nil
}

func (t *Transaction) UpdatePage(page *schema.Page) (int64, error) {
	return t.sess.Update(page)
}

func (t *Transaction) InsertPage(page *schema.Page) (int64, error) {
	return t.sess.Insert(page)
}

func (t *Transaction) GetPendingPageWithTimeoutAndLimit(latestUpdatedAt time.Time, maxNum uint32) ([]*schema.Page, error) {
	var pages []*schema.Page
	err := t.sess.Where("state = ?", enum.PageStatePending).Where("updated_at > ?", latestUpdatedAt).Limit(int(maxNum)).Find(&pages)
	return pages, err
}

func (t *Transaction) GetPendingPageCount() (int64, error) {
	return t.sess.Where("state = ?", enum.PageStatePending).Count()
}
