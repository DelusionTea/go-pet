package conf

import (
	"flag"
	"github.com/caarlos0/env"
	"log"
	"os"
)

const (
	//FilePerm    = 0755
	NumbWorkers = 1
	WorkerBuff  = 100
)

type Config struct {
	//Сервис должен поддерживать конфигурирование следующими методами:
	//адрес и порт запуска сервиса: переменная окружения ОС RUN_ADDRESS или флаг -a;
	//адрес подключения к базе данных: переменная окружения ОС DATABASE_URI или флаг -d;
	//адрес системы расчёта начислений: переменная окружения ОС ACCRUAL_SYSTEM_ADDRESS или флаг -r.

	ServerAddress    string `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	DataBase         string `env:"DATABASE_URI" envDefault:"http://localhost:8080/"`
	SystemAccrualURL string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"localhost:8081/"`
	NumbWorkers      int    `env:"NUMBER_OF_WORKERS"`
	WorkerBuff       int    `env:"WORKERS_BUFFER"`
}

func GetConfig() *Config {
	log.Println("Start Get Config")
	instance := &Config{}
	if err := env.Parse(instance); err != nil {
		log.Fatal(err)
	}
	ServerAddress := flag.String("a", instance.ServerAddress, "run address")
	SystemAccrualURL := flag.String("r", instance.SystemAccrualURL, "accural system address")
	DataBase := flag.String("d", instance.DataBase, "DataBase")
	flag.Parse()

	if os.Getenv("RUN_ADDRESS") == "" {
		instance.ServerAddress = *ServerAddress
	}

	if os.Getenv("SystemAccrualURL") == "" {
		instance.SystemAccrualURL = *SystemAccrualURL
	}
	if os.Getenv("DATABASE_URI") == "" {
		instance.DataBase = *DataBase
	}

	instance.NumbWorkers = NumbWorkers
	instance.WorkerBuff = WorkerBuff

	return instance
}
