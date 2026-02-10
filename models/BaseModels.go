package models

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DBConfig 資料庫配置
type DBConfig struct {
	Hostname string
	Username string
	Password string
	DbName   string
	Port     int
}

// DBManager 讀寫分離的資料庫管理器
type DBManager struct {
	WriteDB *gorm.DB
	ReadDB  *gorm.DB
	SqlDBs  []*sql.DB
}

// PostgresNew 取得讀寫分離的資料庫連線（目前主從共用同一組環境變數）
func PostgresNew() *DBManager {
	// 使用環境變數讀取主庫配置，對應 .env / .env copy
	writeConfig := &DBConfig{
		Hostname: getEnv("POSTGRES_HOST", "localhost"),
		Username: getEnv("POSTGRES_USER", "root"),
		Password: getEnv("POSTGRES_PASSWORD", ""),
		DbName:   getEnv("POSTGRES_DB", "zeabur"),
		Port:     getEnvAsInt("POSTGRES_PORT", 5432),
	}

	// Debug: 印出實際使用的 DB 設定（請確認不要在正式環境留下密碼）
	fmt.Println("[DB Debug] POSTGRES_HOST =", writeConfig.Hostname)
	fmt.Println("[DB Debug] POSTGRES_PORT =", writeConfig.Port)
	fmt.Println("[DB Debug] POSTGRES_USER =", writeConfig.Username)
	fmt.Println("[DB Debug] POSTGRES_DB   =", writeConfig.DbName)

	// 目前讀庫先沿用同一組設定，若之後有讀寫分離再另外拉環境變數
	readConfig := &DBConfig{
		Hostname: writeConfig.Hostname,
		Username: writeConfig.Username,
		Password: writeConfig.Password,
		DbName:   writeConfig.DbName,
		Port:     writeConfig.Port,
	}

	manager, err := NewDBManagerWithReplication(writeConfig, readConfig)
	if err != nil {
		//log.Error("建立資料庫錯誤: %s", err.Error())
		panic(err)
	}
	return manager
}

// NewDBManagerWithReplication 創建讀寫分離的資料庫管理器
func NewDBManagerWithReplication(writeConfig *DBConfig, readConfig *DBConfig) (*DBManager, error) {
	// 連接主庫（寫入）
	writeDSN := buildDSN(writeConfig)
	writeDB, err := gorm.Open(postgres.Open(writeDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("連接主庫失敗: %w", err)
	}

	// 連接從庫（讀取）
	readDSN := buildDSN(readConfig)
	readDB, err := gorm.Open(postgres.Open(readDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("連接從庫失敗: %w", err)
	}

	// 設定連接池
	sqlDBs := make([]*sql.DB, 0)

	// 主庫連接池
	writeSqlDB, err := writeDB.DB()
	if err != nil {
		return nil, fmt.Errorf("無法取得主庫底層 sql.DB: %w", err)
	}
	configureConnectionPool(writeSqlDB)
	sqlDBs = append(sqlDBs, writeSqlDB)

	// 從庫連接池
	readSqlDB, err := readDB.DB()
	if err != nil {
		return nil, fmt.Errorf("無法取得從庫底層 sql.DB: %w", err)
	}
	configureConnectionPool(readSqlDB)
	sqlDBs = append(sqlDBs, readSqlDB)

	return &DBManager{
		WriteDB: writeDB,
		ReadDB:  readDB,
		SqlDBs:  sqlDBs,
	}, nil
}

// buildDSN 構建資料庫連接字符串
func buildDSN(config *DBConfig) string {
	if config.Port == 0 {
		config.Port = 5432
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Taipei",
		config.Hostname, config.Username, config.Password, config.DbName, config.Port)
}

// configureConnectionPool 設定連接池參數
func configureConnectionPool(sqlDB *sql.DB) {
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)
}

// getEnv 讀取字串環境變數，若為空則回傳預設值
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// getEnvAsInt 讀取整數環境變數，若轉換失敗或不存在則回傳預設值
func getEnvAsInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

// GetWrite 獲取寫入資料庫（主庫）
func (m *DBManager) GetWrite() *gorm.DB {
	return m.WriteDB
}

// GetRead 獲取讀取資料庫（指定從庫）
func (m *DBManager) GetRead() *gorm.DB {
	return m.ReadDB // 這裡固定返回單一讀庫
}

// Close 關閉底層 sql 連線
func (m *DBManager) Close() error {
	var firstErr error
	for _, sqlDB := range m.SqlDBs {
		if sqlDB == nil {
			continue
		}
		if err := sqlDB.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// NewPositionIndex 取得下一個可用的 position 索引值
// 如果資料庫中沒有 position 值，則返回 1
// 否則返回當前最大 position 值 + 1
// 此函數會排除 NULL 和無效值（如 0、空字符串等）
// 參數:
//   - model: 模型實例，用於確定查詢的表（例如 &ProductCategory{} 或 &Brand{}）
//
// 返回:
//   - int64: 下一個可用的 position 值
//   - error: 查詢錯誤
func (db *DBManager) NewPositionIndex(model interface{}) (int64, error) {
	var maxPosition *int64
	err := db.GetRead().Model(model).
		Where("position IS NOT NULL AND position > 0").
		Select("MAX(position)").
		Scan(&maxPosition).Error
	if err != nil {
		return 0, fmt.Errorf("查詢最大位置失敗: %w", err)
	}

	// 如果沒有記錄或最大值為 nil（包括 NULL、0、或空值），返回 1
	if maxPosition == nil || *maxPosition <= 0 {
		return 1, nil
	}

	// 返回最大值 + 1
	return *maxPosition + 1, nil
}

// SelectAllModelFields 回傳指定模型所有可更新欄位（排除 id、created_at）
// 這裡透過 DBManager 直接使用內部的 gorm.DB，不需要外部再傳 db 進來
func (db *DBManager) SelectAllModelFields(model interface{}) []string {
	stmt := &gorm.Statement{DB: db.GetWrite()}
	_ = stmt.Parse(model)

	fields := make([]string, 0, len(stmt.Schema.Fields))
	for _, field := range stmt.Schema.Fields {
		// 排除你不想更新的欄位
		if field.Name == "id" || field.DBName == "created_at" {
			continue
		}
		fields = append(fields, field.DBName)
	}

	return fields
}
