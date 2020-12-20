// 注意此处仅提取a标签中href属性的值
// 此外还缺少域名填充
package analyzer

import (
	"context"
	"strings"

	"github.com/andrewyi/crawler/src/entity"
	"github.com/andrewyi/crawler/src/enum"

	"github.com/PuerkitoBio/goquery"
)

type SimpleAnalyzer struct {
	ctx context.Context
}

func NewSimpleAnalyzer(ctx context.Context) Analyzer {

	return &SimpleAnalyzer{
		ctx: ctx,
	}
}

func (a *SimpleAnalyzer) Analyze(page entity.PageInfo) entity.ParsedPageInfo {
	var parsedPageInfo = entity.ParsedPageInfo{
		URL:     page.URL,
		State:   page.State,
		Remark:  page.Remark,
		Content: page.Content,
	}

	if page.State != enum.PageStateSuccess {
		return parsedPageInfo
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(page.Content))
	if err != nil {
		parsedPageInfo.State = enum.PageStateFail
		parsedPageInfo.Remark = err.Error()
		return parsedPageInfo
	}

	var urls map[string]struct{}

	doc.Find("a").Each(func(index int, element *goquery.Selection) {
		href, exists := element.Attr("href")
		if exists {
			urls[href] = struct{}{}
		}
	})
	var subURLs []string
	for u := range urls {
		subURLs = append(subURLs, u)
	}
	parsedPageInfo.SubURLs = subURLs
	return parsedPageInfo
}
