package db

import (
	"bytes"
	"errors"
	"fmt"
	"gorgany/app/core"
	"gorgany/db"
	"gorgany/db/gorm/plugin"
	"gorgany/db/orm"
	"gorgany/internal"
	"gorgany/util"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"
)

const MigrationDir = "db/migration"

type DiffCommand struct {
	modelStructAlreadyAdded map[string]bool
	pivotTables             map[string]bool
}

func (thiz DiffCommand) GetName() string {
	return "db:diff"
}

func (thiz DiffCommand) Execute() {
	thiz.modelStructAlreadyAdded = make(map[string]bool)
	thiz.pivotTables = make(map[string]bool)

	gormDb := db.Builder().GetConnection().Driver().(*gorm.DB)
	tx := gormDb.Begin()
	defer tx.Rollback()

	var statements []string
	tx.Callback().Raw().Register("record_migration", func(tx *gorm.DB) {
		statements = append(statements, tx.Statement.SQL.String())
	})

	moduleName := util.ModuleName()
	modelsMap := internal.GetFrameworkRegistrar().GetDomains()

	pkgInfos, err := util.ScanDir("./pkg/domain")
	if err != nil {
		panic(err)
	}

	for pkgPath, info := range pkgInfos {
		for _, st := range info.Structs {
			if st.FindAnnotationByName("@Embedded") != nil || st.FindAnnotationByName("@Abstract") != nil {
				continue
			}

			key := moduleName + "/" + pkgPath + "." + st.Name
			_, ok := modelsMap[key]
			if !ok {
				fmt.Println("New domain detected, please register it.")
				fmt.Println("Please complete one of the following steps:")
				fmt.Println("- Register it manually, just add it to models registrar(registrar/models.go)")
				fmt.Println("- Run `go run cmd/cli.go domains:register`")
				return
			}
		}
	}

	for _, model := range modelsMap {
		err := thiz.migrateModel(model, tx)
		if err != nil {
			rType := reflect.TypeOf(model)
			fmt.Printf("Domain: %s, error: %v", rType.Name(), err)
			return
		}
	}

	migrator := tx.Migrator()
	for _, model := range modelsMap {
		rType := reflect.TypeOf(model)
		err := thiz.migrateModelConstraints(rType, &statements, migrator)
		if err != nil {
			fmt.Printf("Domain: %s, error: %v", rType.Name(), err)
			return
		}
	}

	thiz.generateMigration(statements)
}

func (thiz DiffCommand) migrateModel(model any, tx *gorm.DB) error {
	migrator := tx.Migrator()

	rModel := util.IndirectType(reflect.TypeOf(model))

	if migrator.HasTable(model) {
		for i := 0; i < rModel.NumField(); i++ {
			field := rModel.Field(i)

			if field.Anonymous && field.Type.Kind() == reflect.Struct && orm.IsParamInTagExists(field.Tag, core.GeneratedDomainTagValue) {
				indirectRModel := util.IndirectType(field.Type)
				rvModel := reflect.New(indirectRModel)
				generatedModel := rvModel.Interface()

				err := thiz.migrateModel(generatedModel, tx)
				if err != nil {
					return err
				}

				continue
			}

			if field.Anonymous || util.IndirectType(field.Type).Kind() == reflect.Struct ||
				util.IndirectType(field.Type).Kind() == reflect.Slice || migrator.HasColumn(model, field.Name) {
				continue
			}

			err := migrator.AddColumn(model, field.Name)
			if err != nil {
				return err
			}
		}
	} else {
		err := migrator.CreateTable(model)
		if err != nil {
			return err
		}
	}

	return thiz.migratePivatTable(model, tx)
}

func (thiz DiffCommand) migratePivatTable(model any, tx *gorm.DB) error {
	namer := schema.NamingStrategy{}
	parseScheme, _ := schema.Parse(model, &sync.Map{}, namer)
	many2manies := parseScheme.Relationships.Many2Many

	for _, relation := range many2manies {
		if _, ok := thiz.pivotTables[relation.JoinTable.Table]; ok {
			continue
		}

		indirectRModel := util.IndirectType(relation.Field.FieldType)
		rvModel := reflect.New(indirectRModel)
		relationModel := rvModel.Interface()

		err := tx.Table(relation.JoinTable.Table).AutoMigrate(model, relationModel)
		if err != nil {
			return err
		}
		thiz.pivotTables[relation.JoinTable.Table] = true
	}
	return nil
}

func (thiz DiffCommand) migrateModelConstraints(rModel reflect.Type, statements *[]string, migrator gorm.Migrator) error {
	namingStrategyService := schema.NamingStrategy{}
	alreadyExtends := false
	for i := 0; i < rModel.NumField(); i++ {
		rField := rModel.Field(i)

		if rField.Anonymous && rField.Type.Kind() == reflect.Struct && orm.IsParamInTagExists(rField.Tag, core.GeneratedDomainTagValue) {
			err := thiz.migrateModelConstraints(rField.Type, statements, migrator)
			if err != nil {
				return err
			}
			continue
		}

		if rField.Anonymous && rField.Type.Kind() == reflect.Struct && orm.IsParamInTagExists(rField.Tag, core.GorganyORMExtends) {
			if alreadyExtends {
				return fmt.Errorf("Gorgany ORM only supports one struct extension!")
			}

			tableName := namingStrategyService.TableName(rField.Type.Name())

			if _, ok := thiz.modelStructAlreadyAdded[tableName]; ok {
				continue
			}

			if !thiz.isColumnExists(tableName, plugin.StructModelColumn()) {
				*statements = append(*statements, fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s varchar(255)", tableName, plugin.StructDefaultColumn))
			}

			alreadyExtends = true

			thiz.modelStructAlreadyAdded[tableName] = true

			err := thiz.migrateModelConstraints(rField.Type, statements, migrator)
			if err != nil {
				return err
			}

			continue
		}

		indirectRModel := util.IndirectType(rModel)
		rvModel := reflect.New(indirectRModel)
		model := rvModel.Interface()

		if migrator.HasConstraint(model, rField.Name) {
			continue
		}

		if err := migrator.CreateConstraint(model, rField.Name); err != nil {
			panic(err)
		}

	}
	return nil
}

func (thiz DiffCommand) isColumnExists(tableName string, columnName string) bool {
	var count int64
	err := db.Builder().Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?", &count, tableName, columnName)
	if err != nil {
		return false
	}
	return count > 0
}

func (thiz DiffCommand) generateMigration(statements []string) {
	if len(statements) == 0 {
		fmt.Println("DB has actual state")
		return
	}

	ddls := make([]string, 0)
	for _, statement := range statements {
		ddls = append(ddls, strings.ReplaceAll(statement, "\"", ""))
	}

	_, callerFilename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(callerFilename)

	content, err := os.ReadFile(filepath.Join(dir, "../../resource/template/command/db_diff.html"))
	if err != nil {
		panic(err)
	}

	tpl, err := template.New("db_diff").Parse(string(content))
	if err != nil {
		panic(err)
	}

	writer := new(bytes.Buffer)

	now := time.Now()
	name := now.Format("20060102_150405.000")
	structName := "Migration" + now.Format("20060102150405")
	fileName := now.Format("20060102150405") + "_migration.go"

	err = tpl.Execute(writer, map[string]any{"Name": name, "StructName": structName, "Statements": ddls})
	if err != nil {
		panic(err)
	}

	if _, err := os.Stat(MigrationDir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(MigrationDir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	err = os.WriteFile(path.Join(MigrationDir, fileName), writer.Bytes(), os.ModePerm)
	if err != nil {
		panic(err)
	}

	fmt.Printf("File %s/%s successfully generated\n", MigrationDir, fileName)
}
