package main

import (
	"build"
	//"fmt"
)

import (
	"github.com/jrick/bitset"
	"math/big"
	"os"
	"path/filepath"
)

var (
	pow256 = BigPow(2, 256)
	cfg    Config
	gPool  *Pool
)

const (
	DiffUnit   = 4 * 1024 * 1024 * 1024 / 256
	UnitChange = float64(4*1024*1024*1024) / 1000 / 1000 / 100 //
)

func PreparePool() {

	LoadConfig("", &cfg)

	//IP := GetPublicIP()
	//day := time.Now().Format("20060102")

	//logFile := "pool_" + cfg.Coin + "_" + IP + "_" + day
	infoFile := "info.log"
	errroFile := "error.log"

	infoFile = filepath.Join("logs", infoFile)
	errroFile = filepath.Join("logs", errroFile)

	blockFile := filepath.Join("logs", "block.log")
	shareFile := filepath.Join("logs", "share.log")
	MkdirIfNoExist("logs")

	InitLog(infoFile, errroFile, shareFile, blockFile)

	Info.Println("Name:", build.BuildName)
	Info.Println("Version:", build.BuildVersion)
	Info.Println("Build Time:", build.BuildTime)
	Info.Println("Go version:", build.GoVersion)
	Info.Println("Commit id:", build.CommitID)
}

func InitPool() {
	cfg.TotalSeconds = cfg.SecondsPerShare * cfg.WindowSize

	gPool = &Pool{Coin: cfg.Coin, IP: GetPublicIP(), version: build.BuildVersion, cfg: cfg}
	gPool.found_block_map = make(map[string]string)

	diff := int64(cfg.StartDiff * DiffUnit)
	gPool.StartTarget = new(big.Int).Div(pow256, new(big.Int).SetInt64(diff))
	gPool.StartDiff4G = cfg.StartDiff / 256
	Info.Println("StartDiff:", cfg.StartDiff, diff, gPool.StartDiff4G, gPool.StartTarget)

	gPool.WalletUrl = cfg.WalletUrl
	gPool.WalletApiUrl = cfg.WalletApiUrl
	gPool.walletReady = make(chan bool)
	gPool.SubmitSol = make(chan Response)

	gPool.miners = make(map[*Miner]struct{})
	gPool.shareSet.shares = make(map[string]struct{})
	gPool.timeout = ParseDuration(cfg.Timeout)

	gPool.Jobs = make(map[string]Job)

	if cfg.ENonceLen < 2 || cfg.ENonceLen > 4 {
		Warning.Printf("Error: ENonceLen:%d out of range [2,3,4]", cfg.ENonceLen)
		os.Exit(0)
	}
	gPool.preEnonce = cfg.PoolId << (cfg.ENonceLen*8 - 4)
	gPool.enonceRange = 1 << (cfg.ENonceLen*8 - 4)
	gPool.enonce = bitset.NewBytes(gPool.enonceRange)
	Info.Println("Init enonce:", gPool.preEnonce)

	gPool.openLevelDB()

	if !gPool.OpenMongoDB() {
		Alert.Fatalln("OpenMongoDB error")
	}
}

func main() {
	PreparePool()
	InitPool()
	go gPool.updateWork()

	<-gPool.walletReady
	go gPool.ListenTCP()

	Info.Println("pool is running...")

	go gPool.PushShareService()
	go gPool.PrintDailyShareServer()
	go gPool.CalcUserReward()
	go gPool.SendRewardToUsers()
	go gPool.CheckTxState()
	gPool.httpServer()

	gPool.closeDB()
	Warning.Println("exit pool.")

}
