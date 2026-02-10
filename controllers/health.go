package controllers

import (
	"net/http"

	"project/models"
	response "project/services/responses"

	"github.com/gin-gonic/gin"
)

// Health 健康檢查
func Health(c *gin.Context) {
	response.New(c).Success("LINE Bot Webhook API is running").Send()
}

// HealthDB 測試資料庫連線
func HealthDB(c *gin.Context) {
	dbm := models.PostgresNew()
	// 簡單 Ping 一下寫庫
	sqlDB, err := dbm.GetWrite().DB()
	if err != nil {
		response.New(c).Fail(http.StatusInternalServerError, "failed to get sql.DB from gorm: "+err.Error()).Send()
		return
	}

	if err := sqlDB.Ping(); err != nil {
		response.New(c).Fail(http.StatusInternalServerError, "database ping failed: "+err.Error()).Send()
		return
	}

	response.New(c).Success("database connection is healthy").Send()
}
