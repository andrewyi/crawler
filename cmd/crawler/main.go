package main

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v1"

	"github.com/andrewyi/crawler/src/server"
)

func main() {

	app := cli.NewApp()

	app.Name = "crawler"
	app.Version = "0.1.0"
	app.Description = "爬虫程序"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config,c",
			Usage: "配置文件",
			Value: "./config.yaml",
		},
	}

	s := server.NewServer()
	app.Action = s.Start

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}
}
