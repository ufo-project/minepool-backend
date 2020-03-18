package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Coin       string `json:"coin"`
	DebugLevel int    `json:"debugLevel"`

	WalletUrl    string `json:"WalletUrl"`
	WalletApiUrl string `json:"WalletApiUrl"`
	ApiKey       string `json:"apikey"`

	MongoDB MongoDb `json:"mongoDB"`
	LevelDB string  `json:"leveldb"`

	PoolIPPort string `json:"PoolIPPort"`

	PoolId    int  `json:"PoolId"`
	ENonceLen uint `json:"ENonceLen"`

	HttpIPPort       string `json:"httpIPPort"`
	LimitHeadersSize int    `json:"limitHeadersSize"`
	LimitBodySize    int64  `json:"limitBodySize"`

	Timeout string `json:"timeout"`

	SecondsPerShare uint `json:"SecondsPerShare"`
	WindowSize      uint `json:"WindowSize"`
	TotalSeconds    uint `json:"TotalSeconds,omitempty"`

	StartDiff float64 `json:"StartDiff"`
	MinDiff   float64 `json:"MinDiff"`
	MaxDiff   float64 `json:"MaxDiff"`

	RewardPeriod    int64  `json:"RewardPeriod"`
	OneBlockReward  int64  `json:"OneBlockReward"`
	SendMinUfo      int64  `json:"SendMinUfo"`
	HalveHeight     int64  `json:"HalveHeight"`
	SendRewardsTime string `json:"SendRewardsTime"`
	ReSendTime string `json:"ReSendTime"`
	PoolFeeRate     int64  `json:"PoolFeeRate"`
	TxFee       int64 `json:"TxFee"`
}

type MongoDb struct {
	Url         string `json:"url"`
	DBname      string `json:"dbname"`
	ShareCol    string `json:"share"`
	BlockCol    string `json:"block"`
	MinerInfo   string `json:"miner"`
	SendTx      string `json:"sendtx"`
	AddrBalance string `json:"balance"`
	User        string `json:"user"`
	Password    string `json:"password"`
}

func LoadConfig(cfgFileName string, cfg *Config) {
	if cfgFileName == "" {
		cfgFileName = "config.json"
	}
	cfgPath, err := filepath.Abs(cfgFileName)
	if err != nil {
		log.Panicln("filepath.Abs ", cfgFileName, " error:", err)
	}
	log.Println("cfgPath:", cfgPath)

	fd, err := os.Open(cfgPath)
	if err != nil {
		log.Panicln("Open ", cfgPath, " error:", err)
	}
	defer fd.Close()

	decoder := json.NewDecoder(fd)

	if err = decoder.Decode(cfg); err != nil {
		log.Panicln("Open ", cfgPath, " error:", err)
	}
}
