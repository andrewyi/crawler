package entity

// 保存了下载的内容
type PageInfo struct {
	URL     string
	State   uint32 // 0/success 1/fail
	Remark  string // error description, if any
	Content string
}

// 保存了分析后的内容，字段与PageInfo一致，只是多了一个解析好的url结合 SubURLs
type ParsedPageInfo struct {
	URL     string
	State   uint32
	Remark  string
	Content string
	SubURLs []string
}
