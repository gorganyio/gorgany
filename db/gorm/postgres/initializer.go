package postgres

import (
	"fmt"
	"github.com/spf13/viper"
	"gorgany/app/core"
	"gorgany/db/gorm/plugin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewGormPostgresConnection(config map[string]any) core.IConnection {
	dsn := getDataSource(config)

	gormConfig := postgres.Config{DSN: dsn, PreferSimpleProtocol: config["prefer_simple_protocol"].(bool)}
	db, err := gorm.Open(postgres.New(gormConfig), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		panic(err)
	}

	db.Logger.LogMode(logger.Info)

	db.Callback().Query().Before("gorm:query").Register("extended_model_processor_add_type_to_where", plugin.ExtendedModelProcessor{}.AddModelTypeToWhere)
	db.Callback().Create().After("gorm:after_create").Register("after_create", plugin.ExtendedModelProcessor{}.AddModelTypeAfterInsert)

	if viper.GetBool("gorm.debug") {
		db = db.Debug()
	}

	return &GormPostgresConnection{gormInstance: db}
}

func getDataSource(config map[string]any) string {
	return fmt.Sprintf("host=%s port=%v user=%s password=%s dbname=%s sslmode=%s",
		config["host"], config["port"], config["username"], config["password"], config["db"], config["ssl"])
}
