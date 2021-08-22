package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const (
	sessionName                 = "isucondition_go"
	conditionLimit              = 20
	frontendContentsPath        = "../public"
	defaultIconFilePath         = "../NoImage.jpg"
	defaultJIAServiceURL        = "http://localhost:5000"
	mysqlErrNumDuplicateEntry   = 1062
	conditionLevelInfo          = "info"
	conditionLevelWarning       = "warning"
	conditionLevelCritical      = "critical"
	scoreConditionLevelInfo     = 3
	scoreConditionLevelWarning  = 2
	scoreConditionLevelCritical = 1
)

var (
	db                  *sqlx.DB
	mySQLConnectionData *MySQLConnectionEnv
)

type Config struct {
	Name string `db:"name"`
	URL  string `db:"url"`
}

type Isu struct {
	ID         int       `db:"id" json:"id"`
	JIAIsuUUID string    `db:"jia_isu_uuid" json:"jia_isu_uuid"`
	Name       string    `db:"name" json:"name"`
	Character  string    `db:"character" json:"character"`
	JIAUserID  string    `db:"jia_user_id" json:"-"`
	CreatedAt  time.Time `db:"created_at" json:"-"`
	UpdatedAt  time.Time `db:"updated_at" json:"-"`
}

type IsuFromJIA struct {
	Character string `json:"character"`
}

type GetIsuListResponse struct {
	ID                 int                      `json:"id"`
	JIAIsuUUID         string                   `json:"jia_isu_uuid"`
	Name               string                   `json:"name"`
	Character          string                   `json:"character"`
	LatestIsuCondition *GetIsuConditionResponse `json:"latest_isu_condition"`
}

type IsuCondition struct {
	ID         int       `db:"id"`
	JIAIsuUUID string    `db:"jia_isu_uuid"`
	Timestamp  time.Time `db:"timestamp"`
	IsSitting  bool      `db:"is_sitting"`
	Condition  string    `db:"condition"`
	Message    string    `db:"message"`
	CreatedAt  time.Time `db:"created_at"`
}

type MySQLConnectionEnv struct {
	Host     string
	Port     string
	User     string
	DBName   string
	Password string
}

type GetIsuJoinedCondition struct {
	ID         int            `db:"id"`
	JIAIsuUUID string         `db:"jia_isu_uuid"`
	Character  string         `db:"character"`
	Name       string         `db:"name"`
	Timestamp  sql.NullTime   `db:"timestamp"`
	IsSitting  sql.NullBool   `db:"is_sitting"`
	Condition  sql.NullString `db:"condition"`
	Message    sql.NullString `db:"message"`
}

type GetIsuConditionResponse struct {
	JIAIsuUUID     string `json:"jia_isu_uuid"`
	IsuName        string `json:"isu_name"`
	Timestamp      int64  `json:"timestamp"`
	IsSitting      bool   `json:"is_sitting"`
	Condition      string `json:"condition"`
	ConditionLevel string `json:"condition_level"`
	Message        string `json:"message"`
}

type TrendResponse struct {
	Character string            `json:"character"`
	Info      []*TrendCondition `json:"info"`
	Warning   []*TrendCondition `json:"warning"`
	Critical  []*TrendCondition `json:"critical"`
}

type TrendCondition struct {
	ID        int   `json:"isu_id"`
	Timestamp int64 `json:"timestamp"`
}

func getEnv(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultValue
}

func NewMySQLConnectionEnv() *MySQLConnectionEnv {
	return &MySQLConnectionEnv{
		Host:     getEnv("MYSQL_HOST", "127.0.0.1"),
		Port:     getEnv("MYSQL_PORT", "3306"),
		User:     getEnv("MYSQL_USER", "isucon"),
		DBName:   getEnv("MYSQL_DBNAME", "isucondition"),
		Password: getEnv("MYSQL_PASS", "isucon"),
	}
}

func (mc *MySQLConnectionEnv) ConnectDB() (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=true&loc=Asia%%2FTokyo", mc.User, mc.Password, mc.Host, mc.Port, mc.DBName)
	return sqlx.Open("mysql", dsn)
}

func init() {
}

func main() {
	e := echo.New()
	e.Debug = true
	e.Logger.SetLevel(log.DEBUG)

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/api/initialize", getInitialize)
	e.GET("/api/isu", getIsuList)
	e.GET("/api/isu-subquery", getIsuListSubquery)

	mySQLConnectionData = NewMySQLConnectionEnv()

	var err error
	db, err = mySQLConnectionData.ConnectDB()
	if err != nil {
		e.Logger.Fatalf("failed to connect db: %v", err)
		return
	}
	db.SetMaxOpenConns(100)
	defer db.Close()

	serverPort := fmt.Sprintf(":%v", getEnv("SERVER_APP_PORT", "3000"))
	e.Logger.Fatal(e.Start(serverPort))
}

func getInitialize(c echo.Context) error {
	tx, err := db.Beginx()
	if err != nil {
		c.Logger().Errorf("db error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer tx.Rollback()
	isuList := []Isu{}
	tx.Exec("TRUNCATE `isu_condition`")
	tx.Select(&isuList, "Select * FROM `isu`")
	for _, isu := range isuList {
		var uuid = isu.JIAIsuUUID
		for i := 0; i < 1000; i++ {
			var isSitting bool
			if i%3 == 0 {
				isSitting = true
			} else {
				isSitting = false
			}
			var message string = fmt.Sprint("this isu is"+uuid+" %d", i)
			var condition string = "is_dirty=true,is_overweight=true,is_broken=false"
			tx.Exec("INSERT INTO `isu_condition`"+
				"	(`jia_isu_uuid`, `timestamp`, `is_sitting`, `condition`, `message`)"+
				"	VALUES (?, ?, ?, ?, ?)",
				uuid, time.Now().Add(-1*time.Duration(i)*time.Hour), isSitting, condition, message)
		}

	}
	return c.JSON(http.StatusOK, isuList)
}

// GET /api/isu
// ISUの一覧を取得
func getIsuList(c echo.Context) error {
	var jiaUserID string = "1"

	tx, err := db.Beginx()
	if err != nil {
		c.Logger().Errorf("db error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer tx.Rollback()

	isuList := []Isu{}
	err = tx.Select(
		&isuList,
		"SELECT * FROM `isu` WHERE `jia_user_id` = ? ORDER BY `id` DESC",
		jiaUserID)
	if err != nil {
		c.Logger().Errorf("db error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	responseList := []GetIsuListResponse{}
	for _, isu := range isuList {
		var lastCondition IsuCondition
		foundLastCondition := true
		err = tx.Get(&lastCondition, "SELECT * FROM `isu_condition` WHERE `jia_isu_uuid` = ? ORDER BY `timestamp` DESC LIMIT 1",
			isu.JIAIsuUUID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				foundLastCondition = false
			} else {
				c.Logger().Errorf("db error: %v", err)
				return c.NoContent(http.StatusInternalServerError)
			}
		}

		var formattedCondition *GetIsuConditionResponse
		if foundLastCondition {
			conditionLevel, err := calculateConditionLevel(lastCondition.Condition)
			if err != nil {
				c.Logger().Error(err)
				return c.NoContent(http.StatusInternalServerError)
			}

			formattedCondition = &GetIsuConditionResponse{
				JIAIsuUUID:     lastCondition.JIAIsuUUID,
				IsuName:        isu.Name,
				Timestamp:      lastCondition.Timestamp.Unix(),
				IsSitting:      lastCondition.IsSitting,
				Condition:      lastCondition.Condition,
				ConditionLevel: conditionLevel,
				Message:        lastCondition.Message,
			}
		}

		res := GetIsuListResponse{
			ID:                 isu.ID,
			JIAIsuUUID:         isu.JIAIsuUUID,
			Name:               isu.Name,
			Character:          isu.Character,
			LatestIsuCondition: formattedCondition}
		responseList = append(responseList, res)
	}

	err = tx.Commit()
	if err != nil {
		c.Logger().Errorf("db error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, responseList)
}

func getIsuListSubquery(c echo.Context) error {
	var jiaUserID string = "1"

	// tx, err := db.Beginx()
	// if err != nil {
	// 	c.Logger().Errorf("db error: %v", err)
	// 	return c.NoContent(http.StatusInternalServerError)
	// }
	// defer tx.Rollback()

	isuConditions := []GetIsuJoinedCondition{}
	err := db.Select(
		&isuConditions,
		"select i.id AS id, i.jia_isu_uuid AS jia_isu_uuid, i.name AS `name`, i.character AS `character`, con.timestamp AS `timestamp`, con.is_sitting AS is_sitting, con.condition AS `condition`, con.message AS message from isu i LEFT JOIN (select * from isu_condition con inner JOIN (select MAX(timestamp) as max_timestamp,jia_isu_uuid as uuid from isu_condition latest group by jia_isu_uuid) latest_con on con.jia_isu_uuid = latest_con.uuid and con.timestamp = latest_con.max_timestamp) con on i.jia_isu_uuid = con.jia_isu_uuid where i.jia_user_id = ?",
		jiaUserID)
	if err != nil {
		c.Logger().Errorf("db error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	responseList := []GetIsuListResponse{}
	for _, isucon := range isuConditions {
		var formattedCondition *GetIsuConditionResponse
		if isucon.Condition.Valid {
			conditionLevel, err := calculateConditionLevel(isucon.Condition.String)
			if err != nil {
				c.Logger().Error(err)
				return c.NoContent(http.StatusInternalServerError)
			}

			formattedCondition = &GetIsuConditionResponse{
				JIAIsuUUID:     isucon.JIAIsuUUID,
				IsuName:        isucon.Name,
				Timestamp:      isucon.Timestamp.Time.Unix(),
				IsSitting:      isucon.IsSitting.Bool,
				Condition:      isucon.Condition.String,
				ConditionLevel: conditionLevel,
				Message:        isucon.Message.String,
			}
		}

		res := GetIsuListResponse{
			ID:                 isucon.ID,
			JIAIsuUUID:         isucon.JIAIsuUUID,
			Name:               isucon.Name,
			Character:          isucon.Character,
			LatestIsuCondition: formattedCondition}
		responseList = append(responseList, res)
	}

	if err != nil {
		c.Logger().Errorf("db error: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, responseList)
}

// ISUのコンディションの文字列からコンディションレベルを計算
func calculateConditionLevel(condition string) (string, error) {
	var conditionLevel string

	warnCount := strings.Count(condition, "=true")
	switch warnCount {
	case 0:
		conditionLevel = conditionLevelInfo
	case 1, 2:
		conditionLevel = conditionLevelWarning
	case 3:
		conditionLevel = conditionLevelCritical
	default:
		return "", fmt.Errorf("unexpected warn count")
	}

	return conditionLevel, nil
}
