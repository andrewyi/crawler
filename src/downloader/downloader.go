package downloader

import (
	"github.com/andrewyi/crawler/src/entity"
)

type Downloader interface {
	Download(string) entity.PageInfo
}
