package main

import (
	"github.com/gin-gonic/gin"
	"github.com/juzeon/poe-openai-proxy/conf"
	"github.com/juzeon/poe-openai-proxy/poe"
	"github.com/juzeon/poe-openai-proxy/router"
	"github.com/juzeon/poe-openai-proxy/util"
	"github.com/robfig/cron/v3"
	"strconv"
)

func main() {
	conf.Setup()
	poe.Setup()
	engine := gin.Default()
	router.Setup(engine)
	c := cron.New()
	_, err := c.AddFunc("@every 30m", poe.CheckClient)
	if err != nil {
		panic(err)
	}
	util.Logger.Info("定时任务启动")
	c.Start()

	err = engine.Run(":" + strconv.Itoa(conf.Conf.Port))
	if err != nil {
		panic(err)
	}
}
