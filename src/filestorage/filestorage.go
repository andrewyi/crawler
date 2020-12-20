package filestorage

import (
	"github.com/andrewyi/crawler/src/entity"
)

type FileStorage interface {
	Store(string, entity.ParsedPageInfo) error
}
