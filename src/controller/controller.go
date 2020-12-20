package controller

import (
	"github.com/andrewyi/crawler/src/entity"
)

type Controller interface {
	Process(entity.ParsedPageInfo) []string
}
