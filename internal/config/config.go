package config

import (
	subpubv1 "github.com/AnikinSimon/vk-test-task/internal/grpc/subpub/v1"
	"log"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Env           string          `mapstructure:"env"`
	GRPCServerCfg subpubv1.Config `mapstructure:"grpc"`
}

func LoadConfig(cfgPath string) (config AppConfig, err error) {
	viper.SetConfigFile(cfgPath)
	viper.MergeInConfig()

	viper.Unmarshal(&config)

	log.Println("config:", config)

	return
}
