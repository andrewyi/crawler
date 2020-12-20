package core

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/andrewyi/crawler/src/dbstorage"
	"github.com/andrewyi/crawler/src/dbstorage/schema"
	"github.com/andrewyi/crawler/src/enum"
	"github.com/andrewyi/crawler/src/util"
)

// 导入seed文件数据，从而启动整个程序运转流程
func CreateSeedRecord(logger *log.Logger, urlQueue chan string, dbStorage *dbstorage.SimpleDBStorage, seedFilePath string) {

	file, err := os.Open(seedFilePath)
	if err != nil {
		logger.WithError(err).Fatal("fail to read seed file")
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var URLs []string

	for scanner.Scan() {
		URLs = append(URLs, scanner.Text())
	}

	t, err := dbStorage.NewTransaction()
	if err != nil {
		logger.WithError(err).Fatal("fail to start transaction")
	}
	defer t.Rollback()

	var toSendURLs []string

	for _, URL := range URLs {
		_, err := t.GetPageWithLock(URL)
		if err != nil {
			if err == dbstorage.ErrDataNotExist { // URL不存在，直接插入
				nURL, err := util.ShortifyURL(URL)
				if err != nil {
					logger.WithError(err).WithField("url", URL).Error("fail to shortify url")
				}
				var paths [][]string = [][]string{
					{nURL},
				}
				pathsStr, err := json.Marshal(paths)
				if err != nil {
					logger.WithError(err).WithField(
						"paths", paths).Fatal("fail to marshal paths into string")
				}
				page := &schema.Page{
					URL:   nURL,
					Paths: string(pathsStr),
				}
				_, err = t.InsertPage(page)
				if err != nil {
					logger.WithError(err).WithField("url", page.URL).Fatal("fail to insert page")
				}

				toSendURLs = append(toSendURLs, URL)

			} else { // query错误，当前处于启动阶段，可以直接报错
				logger.WithError(err).WithField("url", URL).Fatal("fail to get page")

			}
		}
	}
	t.Commit()

	go func() { // 启动新协程发送，防止阻塞主任务
		for _, u := range toSendURLs {
			urlQueue <- u
		}
	}()
}

// 当前仅仅分析超过一定时候仍然处于pending的任务（即没有成功或者失败）
// 并没有重试失败的任务，如果需要重试失败任务，则需要更加清晰定义state，即表明哪些错误是可以重试的，哪些又不可以
func CreateRetryTask(ctx context.Context, logger *log.Logger, urlQueue chan string, dbStorage *dbstorage.SimpleDBStorage, scanPeriod uint32, taskTimeout uint32) {

	timer := time.NewTimer(time.Second * time.Duration(scanPeriod))
	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			case <-timer.C:
				//dowork
				RetryTask(logger, urlQueue, dbStorage, taskTimeout)
			}
		}
	}()

}

func RetryTask(logger *log.Logger, urlQueue chan string, dbStorage *dbstorage.SimpleDBStorage, taskTimeout uint32) {

	t, err := dbStorage.NewTransaction()
	if err != nil {
		logger.WithError(err).Error("fail to start transaction")
		return
	}
	defer t.Rollback()

	flagTime := time.Now().Add(-1 * time.Second * time.Duration(taskTimeout))

	pages, err := t.GetPendingPageWithTimeoutAndLimit(flagTime, enum.MaxRetryTaskNum)
	if err != nil {
		logger.WithError(err).Error("fail to get pages")
		return
	}

	go func() {
		for _, p := range pages {
			urlQueue <- p.URL
		}
	}()
}

// 当前判断程序运行结束的方式为：
// 定时扫描所有的url信息，判断如果已经没有pending的url，则任务运行结束
// TODO: 事实上当存储发生sharding时，这个查询就变得非常困难，因此还需要持续优化，但是目前没有想到优化方式
func CreateCheckCompletedTask(ctx context.Context, logger *log.Logger, dbStorage *dbstorage.SimpleDBStorage, checkPeriod uint32, finished chan struct{}) {

	timer := time.NewTimer(time.Second * time.Duration(checkPeriod))
	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			case <-timer.C:
				//dowork
				CheckTask(logger, dbStorage, finished)
			}
		}
	}()
}

func CheckTask(logger *log.Logger, dbStorage *dbstorage.SimpleDBStorage, finished chan struct{}) {
	t, err := dbStorage.NewTransaction()
	if err != nil {
		logger.WithError(err).Error("fail to start transaction")
		return
	}
	defer t.Rollback()
	count, err := t.GetPendingPageCount()
	if err != nil {
		logger.WithError(err).Error("fail to get pending page count")
		return
	}

	if count == 0 {
		finished <- struct{}{}
	}

}
