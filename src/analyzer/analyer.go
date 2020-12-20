package analyzer

import (
	"github.com/andrewyi/crawler/src/entity"
)

type Analyzer interface {
	Analyze(entity.PageInfo) entity.ParsedPageInfo
}
