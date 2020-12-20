package controller

import (
	"context"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/andrewyi/crawler/src/dbstorage"
	"github.com/andrewyi/crawler/src/dbstorage/schema"
	"github.com/andrewyi/crawler/src/entity"
	"github.com/andrewyi/crawler/src/enum"
	"github.com/andrewyi/crawler/src/filestorage"
	"github.com/andrewyi/crawler/src/util"
)

type SimpleController struct {
	ctx      context.Context
	logger   *log.Logger
	location string
	depth    uint8

	file filestorage.FileStorage
	db   *dbstorage.SimpleDBStorage
}

func NewSimpleController(ctx context.Context, depth uint8, location string, dbStorage *dbstorage.SimpleDBStorage, logger *log.Logger) Controller {

	var c = &SimpleController{
		ctx:      ctx,
		logger:   logger,
		location: location,
		depth:    depth,
	}

	f := filestorage.NewSimpleFileStorage(ctx, c.location)
	c.file = f
	c.db = dbStorage
	return c
}

func (c *SimpleController) Process(parsedPage entity.ParsedPageInfo) []string {

	domain, err := util.GetDomain(parsedPage.URL)
	if err != nil {
		// 非致命错误，直接返回
		c.logger.WithError(err).WithField("domain", domain).Error("fail to parse url domain")
		return nil
	}

	nURL, err := util.ShortifyURL(parsedPage.URL)
	if err != nil {
		// 非致命错误，直接返回。打印错误日志，后续需要人工介入处理
		c.logger.WithError(err).WithField("url", parsedPage.URL).Error("fail to shortify url")
		return nil
	}

	t, err := c.db.NewTransaction()
	if err != nil {
		// 非致命错误，直接返回。后续操作交给重试机制
		c.logger.WithError(err).WithField("url", parsedPage.URL).Error("fail to start transaction")
		return nil
	}
	defer t.Rollback()

	// 获取url在数据库中对应的记录，记录必须存在
	page, err := t.GetPageWithLock(nURL) // 当前记录被锁定，简单避免并发问题，但少许影响性能
	if err != nil {
		// 非致命错误，直接返回。打印错误日志，后续需要人工介入分析
		// 逻辑上不存在数据库中找不到记录的情况（都会事前创建），如果是连接错误则直接重试即可
		c.logger.WithError(err).WithField("url", nURL).Error("fail to find page")
		return nil
	}

	// 如果数据库中记录已经为成功处理，则直接返回即可
	if page.State == enum.PageStateSuccess {
		c.logger.WithField("url", nURL).Info("url has been processed")
		return nil
	}

	if parsedPage.State == enum.PageStateFail {
		// 更新为失败状态
		page.State = enum.PageStateFail
		page.Remark = parsedPage.Remark
		_, err = t.UpdatePage(page)
		if err != nil {
			c.logger.WithError(err).WithField("url", nURL).Info("update failed")
			return nil
		}

		t.Commit()
		return nil
	}

	// 更新为成功状态
	page.State = enum.PageStateSuccess
	page.FetchedAt = time.Now()
	subURLsString, err := json.Marshal(parsedPage.SubURLs)
	if err != nil {
		c.logger.WithError(err).WithField(
			"sub urls", parsedPage.SubURLs).Error("fail to marshal sub urls into string")
		return nil
	}
	page.SubURLs = string(subURLsString)
	_, err = t.UpdatePage(page)
	if err != nil {
		c.logger.WithError(err).WithField("url", nURL).Info("update failed")
		return nil
	}

	// 执行文件系统存储
	err = c.file.Store(domain, parsedPage) // 可以安全重试
	if err != nil {
		// TODO: 区分文件存储的致命错误（例如磁盘空间不足、权限问题）
		// 当前视为非致命错误，返回并继续
		c.logger.WithError(err).WithField("domain", domain).Error("fail to store url content")
		return nil
	}

	// 继续处理后续任务（subURLs），注意如果爬取失败则直接忽略
	// 转SubURLs为set
	var toCheckSubURLs = make(map[string]struct{})
	for _, s := range parsedPage.SubURLs {
		toCheckSubURLs[s] = struct{}{}
	}
	subURLs, err := c.ProcessSubURLs(t, page, toCheckSubURLs)
	if err != nil {
		c.logger.WithError(err).WithField("url", nURL).Error("fail to process sub url records")
		return nil

	} else {
		t.Commit() // 最终无误后commit
	}

	return subURLs
}

func (c *SimpleController) ProcessSubURLs(
	t *dbstorage.Transaction, page *schema.Page, subURLs map[string]struct{}) ([]string, error) {

	// 提取page的path中所有url
	var URLPaths [][]string
	err := json.Unmarshal([]byte(page.Paths), &URLPaths)
	if err != nil {
		c.logger.WithError(err).WithField("paths", page.Paths).Error("fail to unmarshal paths")
		return nil, err
	}

	var shortPathDepth bool

	for _, p := range URLPaths {
		if len(p) < int(c.depth) {
			shortPathDepth = true
			break
		}
	}

	if !shortPathDepth { // 所有路径的长度都达到或超过depth，无需继续爬取
		return nil, nil
	}

	for _, p1 := range URLPaths {
		for _, p2 := range p1 {
			delete(subURLs, p2) // 移除已经在路径中的subURL
		}
	}

	var toProcessSubURLs = make(map[string]struct{})

	// 开始针对每一个subURL判断是否需要进行爬取
	for subURL := range subURLs {
		nURL, err := util.ShortifyURL(subURL)
		if err != nil {
			c.logger.WithError(err).WithField("url", subURL).Error("fail to shortify url")
			return nil, nil
		}
		nPage, err := t.GetPageWithLock(nURL) // 当前记录被锁定，简单避免并发问题，但少许影响性能
		if err != nil {
			if err == dbstorage.ErrDataNotExist { // nURL不存在，直接插入
				var nPageURLPaths [][]string
				for _, path := range URLPaths {
					path = append(path, nURL)
					nPageURLPaths = append(nPageURLPaths, path)
				}
				pathsStr, err := json.Marshal(nPageURLPaths)
				if err != nil {
					c.logger.WithError(err).WithField(
						"paths", nPageURLPaths).Error("fail to marshal paths into string")
					return nil, err
				}
				nPage = &schema.Page{
					URL:   nURL,
					Paths: string(pathsStr),
				}
				_, err = t.InsertPage(nPage)
				if err != nil {
					c.logger.WithError(err).WithField("url", nPage.URL).Error("fail to insert page")
					return nil, err
				}

				toProcessSubURLs[subURL] = struct{}{}
				continue

			} else {
				c.logger.WithError(err).WithField("url", subURL).Error("fail to create page")
				return nil, err

			}
		}

		// 如果成功获取到npage，则需要进一步处理
		// 首先将当前path的路径加入到此下一级npage中
		var nPageURLPaths [][]string
		err = json.Unmarshal([]byte(nPage.Paths), &nPageURLPaths)
		if err != nil {
			c.logger.WithError(err).WithField("paths", nPage.Paths).Error("fail to unmarshal paths")
			return nil, err
		}

		for _, path := range URLPaths {
			path = append(path, nURL)
			var exists bool
			for _, nPath := range nPageURLPaths {
				if util.StringSliceEqual(nPath, path) {
					exists = true
					break
				}
			}
			if !exists {
				nPageURLPaths = append(nPageURLPaths, path)
			}
		}
		nPagePathsStr, err := json.Marshal(nPageURLPaths)
		if err != nil {
			c.logger.WithError(err).WithField(
				"sub urls", nPagePathsStr).Error("fail to marshal sub urls into string")
			return nil, err
		}
		nPage.Paths = string(nPagePathsStr)
		_, err = t.UpdatePage(nPage)
		if err != nil {
			c.logger.WithError(err).WithField("url", nURL).Info("update failed")
			return nil, err
		}

		// 如果当前npage为成功结束状态，则还应继续深入分析此npage对应的下一级subURLs
		// 正好是一个递归
		if nPage.State == enum.PageStateSuccess {
			var nPageSubURLs []string
			err := json.Unmarshal([]byte(nPage.SubURLs), &nPageSubURLs)
			if err != nil {
				c.logger.WithError(err).WithField("sub_urls", page.Paths).Error("fail to unmarshal sub urls")
				return nil, err
			}
			var nSubURLs = make(map[string]struct{})
			for _, u := range nPageSubURLs {
				nSubURLs[u] = struct{}{}
			}

			us, err := c.ProcessSubURLs(t, nPage, nSubURLs)
			if err != nil {
				c.logger.WithError(err).WithField("url", nPage.URL).Error("fail to process sub url records")
				return nil, err
			}
			for _, k := range us {
				toProcessSubURLs[k] = struct{}{}
			}
		}

	}

	var toReturn []string
	for k := range toProcessSubURLs {
		toReturn = append(toReturn, k)
	}

	return toReturn, nil
}
