package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"linebot/controller"
	"linebot/middleware"
	"linebot/route"
	linebotsvc "linebot/service/linebot"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	// ENV 決定吃哪個 config：本地設 ENV=dev 用 config_dev.yaml，部署時由雲端設 ENV=prod/dev
	env := os.Getenv("ENV")
	if env == "" {
		env = "prod"
	}
	configPath := filepath.Join("config", "config_"+env+".yaml")

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	_ = v.BindEnv("Server.LINE_CHANNEL_SECRET", "LINE_CHANNEL_SECRET")
	_ = v.BindEnv("Server.LINE_CHANNEL_ACCESS_TOKEN", "LINE_CHANNEL_ACCESS_TOKEN")

	var channelSecret, channelToken string
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("未找到 config 檔 (%s)，改從環境變數讀取", configPath)
			channelSecret = os.Getenv("LINE_CHANNEL_SECRET")
			channelToken = os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
		} else {
			log.Fatalf("讀取設定檔: %v", err)
		}
	} else {
		// viper BindEnv：環境變數會自動覆寫 YAML 同名字
		channelSecret = v.GetString("Server.LINE_CHANNEL_SECRET")
		channelToken = v.GetString("Server.LINE_CHANNEL_ACCESS_TOKEN")
	}

	if channelSecret == "" || channelToken == "" {
		log.Fatal("LINE_CHANNEL_SECRET 與 LINE_CHANNEL_ACCESS_TOKEN 必須在 config 或環境變數中設定")
	}

	lineService, err := linebotsvc.NewLineBotService(channelSecret, channelToken)
	if err != nil {
		log.Fatal(err)
	}
	lineCtrl := controller.NewLineController(lineService)

	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "::1"})

	r.Use(middleware.LineControllerMiddleware(lineCtrl))
	route.Setup(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	log.Printf("LINE Bot 服務啟動於 port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("HTTP 服務錯誤: %v", err)
	}
}
