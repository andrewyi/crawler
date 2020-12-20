// 仅仅实现了简单的http Get方式下载
// TODO:
// 1. 增加更多GET配置，包括agent、cookies、proxy、content-type
// 2. 增加更多的错误验证(http response status)
// 3. 支持http/https，细化cname/3xx跳转
package downloader

import (
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/andrewyi/crawler/src/entity"
	"github.com/andrewyi/crawler/src/enum"
)

type SimpleDownloader struct {
	ctx     context.Context
	timeout uint32
	retry   uint32

	client *http.Client
}

func NewSimpleDownloader(ctx context.Context, timeout uint32, retry uint32) Downloader {

	return &SimpleDownloader{
		ctx:     ctx,
		timeout: timeout,
		retry:   retry,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

func (s *SimpleDownloader) Download(url string) entity.PageInfo {
	var (
		err        error
		retryCount uint32
		content    []byte
	)

	for {
		if retryCount >= s.retry {
			break
		}

		resp, err := s.client.Get(url)
		if err != nil {
			retryCount++
			continue
		}
		defer resp.Body.Close()
		content, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			retryCount++
			continue
		}

		if err == nil {
			break
		}
	}
	if err != nil {
		return entity.PageInfo{
			URL:    url,
			State:  enum.PageStateFail,
			Remark: err.Error(),
		}
	}

	return entity.PageInfo{
		URL:     url,
		State:   enum.PageStateSuccess,
		Content: string(content),
	}
}
