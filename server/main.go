package main

import (
	"fmt"

	"os"
	"strconv"

	"github.com/7574-sistemas-distribuidos/docker-compose-init/server/common"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

const PATH_CONFIG = "./config.ini"

type ConfigParams struct {
	Port          int
	ListenBacklog int
	LoggingLevel  string
}

func initializeConfig() (ConfigParams, error) {
	v := viper.New()

	// ENV
	v.AutomaticEnv()

	// Leer config.ini
	v.SetConfigFile("./config.ini")
	v.SetConfigType("ini")

	if err := v.ReadInConfig(); err != nil {
		fmt.Println("No config file found, using env vars")
	}

	// Validaciones
	if v.GetInt("DEFAULT.SERVER_PORT") == 0 {
		return ConfigParams{}, fmt.Errorf("missing SERVER_PORT")
	}

	if v.GetInt("DEFAULT.SERVER_LISTEN_BACKLOG") == 0 {
		return ConfigParams{}, fmt.Errorf("missing SERVER_LISTEN_BACKLOG")
	}

	if v.GetString("DEFAULT.LOGGING_LEVEL") == "" {
		return ConfigParams{}, fmt.Errorf("missing LOGGING_LEVEL")
	}

	return ConfigParams{
		Port:          v.GetInt("DEFAULT.SERVER_PORT"),
		ListenBacklog: v.GetInt("DEFAULT.SERVER_LISTEN_BACKLOG"),
		LoggingLevel:  v.GetString("DEFAULT.LOGGING_LEVEL"),
	}, nil
}

func InitLogger(logLevel string) error {
	baseBackend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(
		`%{time:2006-01-02 15:04:05} %{level:.5s}     %{message}`,
	)
	backendFormatter := logging.NewBackendFormatter(baseBackend, format)

	backendLeveled := logging.AddModuleLevel(backendFormatter)
	logLevelCode, err := logging.LogLevel(logLevel)
	if err != nil {
		return err
	}
	backendLeveled.SetLevel(logLevelCode, "")

	logging.SetBackend(backendLeveled)
	return nil
}

func main() {
	configParams, err := initializeConfig()
	if err != nil {
		log.Critical(err)
	}

	loggingLevel := configParams.LoggingLevel
	port := configParams.Port
	listenBacklog := configParams.ListenBacklog

	InitLogger(loggingLevel)

	log.Debugf(
		"action: config | result: success | port: %d | listen_backlog: %d | logging_level: %s",
		port,
		listenBacklog,
		loggingLevel,
	)

	server, err := common.NewServer(strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}

	server.Run()
}
