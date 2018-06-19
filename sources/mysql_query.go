package sources

import (
	"database/sql"
	_ "github.com/siddontang/go-mysql/driver"
	log "github.com/sirupsen/logrus"
	"strconv"

	"encoding/json"
	"github.com/Pirionfr/lookatch-common/control"
)

const MysqlQueryType = "MysqlQuery"

type (
	MySQLQuery struct {
		*JDBCQuery
		config MysqlQueryConfig
	}

	MysqlQueryConfig struct {
		*JDBCQueryConfig
		Schema  string   `json:"schema"`
		Exclude []string `json:"exclude"`
	}
)

func newMysqlQuery(s *Source) (SourceI, error) {
	jdbcQuery := NewJDBCQuery(s)

	mysqlQueryConfig := MysqlQueryConfig{}
	s.Conf.UnmarshalKey("sources."+s.Name, &mysqlQueryConfig)
	mysqlQueryConfig.JDBCQueryConfig = &jdbcQuery.Config

	return &MySQLQuery{
		JDBCQuery: &jdbcQuery,
		config:    mysqlQueryConfig,
	}, nil
}

func (m *MySQLQuery) Init() {

	//start bi Query Schema
	err := m.QuerySchema()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error while querying Schema")
		return
	}
	log.Debug("Init Done")
}

func (m *MySQLQuery) GetStatus() interface{} {
	m.Connect("information_schema")
	defer m.db.Close()
	return m.JDBCQuery.GetStatus()
}

func (m *MySQLQuery) HealtCheck() bool {
	m.Connect("information_schema")
	defer m.db.Close()
	return m.JDBCQuery.HealtCheck()
}

func (m *MySQLQuery) Connect(schema string) {

	dsn := m.config.User + ":" + m.config.Password + "@" + m.config.Host + ":" + strconv.Itoa(m.config.Port) + "?" + schema

	//first check if db is not already established
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("open mysql connection")
	} else {
		m.db = db
	}

	err = m.db.Ping()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Connection is dead")
	}

}

func (m *MySQLQuery) Process(action string, params ...interface{}) interface{} {

	switch action {
	case control.SourceQuery:
		evSqlQuery := &Query{}
		payload := params[0].([]byte)
		err := json.Unmarshal(payload, evSqlQuery)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Unable to unmarshal MySQL Query Statement event")
		} else {
			m.Query(evSqlQuery.Query)
		}
		break
	default:
		log.WithFields(log.Fields{
			"action": action,
		}).Error("action not implemented")
	}
	return nil
}

func (m *MySQLQuery) QuerySchema() (err error) {

	m.Connect("information_schema")
	defer m.db.Close()

	excluded := m.config.Exclude
	notin := "'information_schema','mysql','performance_schema','sys'"

	//see which tables are excluded
	for _, dbname := range excluded {
		notin = notin + ",'" + dbname + "'"
	}
	log.Info("exclude:", notin)

	q := "SELECT TABLE_CATALOG ,TABLE_SCHEMA ,TABLE_NAME, COLUMN_NAME, ORDINAL_POSITION, IS_NULLABLE, DATA_TYPE, " +
		"CHARACTER_MAXIMUM_LENGTH, NUMERIC_PRECISION, NUMERIC_SCALE, COLUMN_TYPE, COLUMN_KEY FROM COLUMNS " +
		"WHERE TABLE_SCHEMA NOT IN (" + notin + ") ORDER BY TABLE_NAME"

	m.JDBCQuery.QuerySchema(q)

	return
}

func (m *MySQLQuery) Query(query string) {
	m.Connect("information_schema")
	defer m.db.Close()
	m.JDBCQuery.Query("", query)
}

func (m *MySQLQuery) QueryMeta(query string, table string, db string, mapAdd map[string]interface{}) map[string]interface{} {
	m.Connect("information_schema")
	defer m.db.Close()
	return m.JDBCQuery.QueryMeta(query, table, db, mapAdd)

}
