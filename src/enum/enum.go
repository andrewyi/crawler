package enum

const (
	// 定义了page的状态
	// 当前不会针对已经下载好、或下载失败的页面再次进行下载，即只操作一次，后续可以优化
	PageStatePending = 0
	PageStateSuccess = 1
	PageStateFail    = 2

	MaxRetryTaskNum = 10
)
