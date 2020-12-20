package filestorage

import (
	"context"
	"os"
	"path/filepath"

	"github.com/andrewyi/crawler/src/entity"
	"github.com/andrewyi/crawler/src/util"
)

type SimpleFileStorage struct {
	ctx      context.Context
	location string
}

func NewSimpleFileStorage(ctx context.Context, location string) FileStorage {

	return &SimpleFileStorage{
		ctx:      ctx,
		location: location,
	}
}

// 以domain作为sharding key来建立文件夹，防止单一文件夹中包含文件数量过多
// 可以将dir的获取作为函数，以创建更加复杂的sharding逻辑
func (s *SimpleFileStorage) Store(domain string, parsedPage entity.ParsedPageInfo) error {
	dir := filepath.Join(s.location, domain)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		if os.IsExist(err) {
			err = nil // ignore
		} else {
			return err
		}
	}

	fileName, err := util.ShortifyURL(parsedPage.URL)
	if err != nil {
		return err
	}
	fp := filepath.Join(dir, fileName)

	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(parsedPage.Content)
	return err
}
