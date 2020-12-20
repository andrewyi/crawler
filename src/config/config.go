package config

type Config struct {
	Log struct {
		Context bool   `mapstructure:"context"`
		Level   string `mapstructure:"level"`
	} `mapstructure:"log"`

	Core struct {
		URLQueueSize            uint32 `mapstructure:"url_queue_size"`
		PageInfoQueueSize       uint32 `mapstructure:"page_info_queue_size"`
		ParsedPageInfoQueueSize uint32 `mapstructure:"parsed_page_info_queue_size"`
		SeedFilePath            string `mapstructure:"seed_file_path"`
		RetryTaskScanPeriod     uint32 `mapstructure:"retry_task_scan_period"`
		TaskTimeout             uint32 `mapstructure:"task_timeout"`
		CheckCompletedPeriod    uint32 `mapstructure:"check_completed_period"`
	} `mapstructure:"log"`

	Database struct {
		URL string `mapstructure:"url"`
	} `mapstructure:"database"`

	Storage struct {
		Location string `mapstructure:"location"`
	} `mapstructure:"storage"`

	Downloader struct {
		Worker  uint32 `mapstructure:"worker"`
		Timeout uint32 `mapstructure:"timeout"`
		Retry   uint32 `mapstructure:"retry"`
	} `mapstructure:"downloader"`

	Analyzer struct {
		Worker uint32 `mapstructure:"worker"`
	} `mapstructure:"analyzer"`

	Controller struct {
		Worker uint32 `mapstructure:"worker"`
		Depth  uint8  `mapstructure:"depth"`
	} `mapstructure:"controller"`
}
