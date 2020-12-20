package util

import (
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

func ReadConfig(filePath string, out interface{}) error {
	v := viper.New()
	v.SetConfigFile(filePath)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // for nested structure
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return err
	}

	if err := v.Unmarshal(&out); err != nil {
		return err
	}

	return nil
}

// 将url的协议、hash tag字段移除
func ShortifyURL(u string) (string, error) {
	oURL, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	oURL.Scheme = ""
	oURL.Fragment = ""
	return oURL.String(), nil
}

// host中可能残留有:port信息，需要进一步移除
func GetDomain(u string) (string, error) {
	oURL, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return strings.Split(oURL.Host, ":")[0], nil
}

// string slice equal
func StringSliceEqual(s1 []string, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i, v := range s1 {
		if v != s2[i] {
			return false
		}
	}
	return true
}
