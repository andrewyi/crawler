package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"

	"github.com/andrewyi/crawler/src/analyzer"
	"github.com/andrewyi/crawler/src/config"
	"github.com/andrewyi/crawler/src/controller"
	"github.com/andrewyi/crawler/src/core"
	"github.com/andrewyi/crawler/src/dbstorage"
	"github.com/andrewyi/crawler/src/downloader"
	"github.com/andrewyi/crawler/src/entity"
	"github.com/andrewyi/crawler/src/routingpool"
	"github.com/andrewyi/crawler/src/util"
)

type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
	logger *log.Logger
	config *config.Config

	urlQueue        chan string
	pageQueue       chan entity.PageInfo
	parsedPageQueue chan entity.ParsedPageInfo

	downloader routingpool.RoutingPool
	analyzer   routingpool.RoutingPool
	controller routingpool.RoutingPool

	dbStorage *dbstorage.SimpleDBStorage

	finished chan struct{}
}

func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		ctx:      ctx,
		cancel:   cancel,
		finished: make(chan struct{}),
	}
}

func (s *Server) initLog() {
	var logger = log.New()
	logger.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	logger.SetOutput(os.Stdout)

	if s.config.Log.Context {
		logger.SetReportCaller(true)
	}

	if logLevel, err := log.ParseLevel(s.config.Log.Level); err != nil {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(logLevel)
	}
	s.logger = logger
}

func (s *Server) Start(ctx *cli.Context) error {
	var err error

	configPath := ctx.String("config")
	var cfg = &config.Config{}
	if err = util.ReadConfig(configPath, cfg); err != nil {
		return fmt.Errorf("fail to load config, err: %w", err)
	}
	s.config = cfg

	s.initLog()

	// downloader从中读取url
	urlQueue := make(chan string, cfg.Core.URLQueueSize)
	// downloader下载的内容将被放入此queue，并由analyzer读取
	pageQueue := make(chan entity.PageInfo, cfg.Core.PageInfoQueueSize)
	// analyzer分析好的内容将被放入此queue，并由controller读取
	parsedPageQueue := make(chan entity.ParsedPageInfo, cfg.Core.ParsedPageInfoQueueSize)

	s.downloader = routingpool.NewSimpleRoutingPool(
		s.ctx,
		cfg.Downloader.Worker,
		func(ctx context.Context) {
			d := downloader.NewSimpleDownloader(ctx, cfg.Downloader.Timeout, cfg.Downloader.Retry)
			for {
				select {
				case <-ctx.Done():
					break
				case url := <-urlQueue:
					page := d.Download(url)
					pageQueue <- page
				}
			}
		},
	)

	s.analyzer = routingpool.NewSimpleRoutingPool(
		s.ctx,
		cfg.Analyzer.Worker,
		func(ctx context.Context) {
			a := analyzer.NewSimpleAnalyzer(ctx)
			for {
				select {
				case <-ctx.Done():
					break
				case page := <-pageQueue:
					parsedPage := a.Analyze(page)
					parsedPageQueue <- parsedPage
				}
			}
		},
	)

	// 不仅controller需要用到dbstorage，后续的相关也需要此handler
	dbStorage, err := dbstorage.NewSimpleDBStorage(cfg.Database.URL)
	if err != nil {
		// 无法恢复的灾难，直接终止
		s.logger.WithError(err).Fatal("fail to create dbstorage handler")
	}
	s.dbStorage = dbStorage

	s.controller = routingpool.NewSimpleRoutingPool(
		s.ctx,
		cfg.Controller.Worker,
		func(ctx context.Context) {
			c := controller.NewSimpleController(ctx, cfg.Controller.Depth, cfg.Storage.Location, dbStorage, s.logger)
			for {
				select {
				case <-ctx.Done():
					break
				case parsedPage := <-parsedPageQueue:
					// 与其他queue 1:1的请求/结果不同，这里一个请求对应多个结果（解析出多个sub url）
					urls := c.Process(parsedPage)
					for _, u := range urls {
						urlQueue <- u
					}
				}
			}
		},
	)

	err = s.downloader.Start()
	if err != nil {
		s.logger.WithError(err).Fatal("fail to start downloader")
	}

	err = s.analyzer.Start()
	if err != nil {
		s.logger.WithError(err).Fatal("fail to start analyzer")
	}

	err = s.controller.Start()
	if err != nil {
		s.logger.WithError(err).Fatal("fail to start controller")
	}

	// 注入seed url数据
	core.CreateSeedRecord(s.logger, urlQueue, dbStorage, cfg.Core.SeedFilePath)

	// 设置重试任务
	core.CreateRetryTask(s.ctx, s.logger, urlQueue, dbStorage, cfg.Core.RetryTaskScanPeriod, cfg.Core.TaskTimeout)

	// 设置终止探测
	core.CreateCheckCompletedTask(s.ctx, s.logger, dbStorage, cfg.Core.CheckCompletedPeriod, s.finished)

	s.wait()
	s.Stop()

	return nil
}

func (s *Server) wait() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT)
	select {
	case <-c:
		s.logger.Warn("interrupt signal, server gonna stop")
		break
	case <-s.finished: // 这是checkcompleted任务触发的，认为当前所有的page都已经被抓取，可以终止程序
		s.logger.Info("task finished, server gonna stop")
		break
	}
}

func (s *Server) Stop() {
	s.cancel()
	s.downloader.Stop()
	s.analyzer.Stop()
	s.controller.Stop()
	s.dbStorage.Close()
}
