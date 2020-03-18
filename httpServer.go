package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type IPResp struct {
	Username    string            `json:"username"`
	WorkerCount int               `json:"workerCount"`
	WorkerIP    map[string]string `json:"workerIP"`
}

func QueryIPHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	var username, workername string
	var isWorker bool

	vars := mux.Vars(req)
	if name, isFound := vars["name"]; isFound {
		names := strings.SplitN(name, ".", 2)
		if len(names) > 0 {
			username = names[0]
			username = strings.ToLower(username)
			if len(names) == 2 {
				workername = names[1]
				isWorker = true
				Info.Println("QueryIPHandler username:", username, "workername:", workername)
			} else {
				Info.Println("QueryIPHandler username:", username)
			}
		} else {
			return
		}

		workerIP := make(map[string]string)
		var ipResp IPResp
		ipResp.WorkerIP = workerIP
		ipResp.Username = username

		gPool.minersLock.RLock()
		for miner, _ := range gPool.miners {
			if miner.username == username {
				if isWorker {
					if miner.workername == workername {
						workerIP[workername] = miner.IP
						ipResp.WorkerCount++
					}
				} else {
					workerIP[miner.workername] = miner.IP
					ipResp.WorkerCount++
				}

			}
		}
		gPool.minersLock.RUnlock()

		workerIPJson, err := json.Marshal(&ipResp)
		if err != nil {
			Warning.Println("workerIPJson Marshal error:", err)
		}

		workerIPStr := string(workerIPJson) + "\n"
		n, err := res.Write([]byte(workerIPStr))
		if err != nil {
			Warning.Println("workerIPJson Write error:", err)
		} else if n != len(workerIPStr) {
			Warning.Printf("workerIPJson Write %d!=%d", n, len(workerIPStr))
		}
	}
}

type Worker struct {
	Username    string   `json:"username"`
	WorkerCount int      `json:"workerCount"`
	Workers     []string `json:"workers"`
}

func QueryWorkerHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	var worker Worker

	vars := mux.Vars(req)
	if username, isFound := vars["name"]; isFound {
		username = strings.ToLower(username)
		worker.Username = username
		Info.Println("QueryWorkerHandler username:", username)

		gPool.minersLock.RLock()
		for miner, _ := range gPool.miners {
			if miner.username == username {
				worker.Workers = append(worker.Workers, miner.workername)
				worker.WorkerCount++
			}
		}
		gPool.minersLock.RUnlock()

		workerJson, err := json.Marshal(&worker)
		if err != nil {
			Warning.Println("workerJson Marshal error:", err)
		}

		workerStr := string(workerJson) + "\n"
		n, err := res.Write([]byte(workerStr))
		if err != nil {
			Warning.Println("workerJson Write error:", err)
		} else if n != len(workerStr) {
			Warning.Printf("workerJson Write %d!=%d", n, len(workerStr))
		}
	}
}

type WorkerUsers struct {
	Workername    string   `json:"workername"`
	UserCount int      `json:"userCount"`
	Users     []string `json:"users"`
}

func StringsContains(array []string, val string) (index int) {
	index = -1
	for i := 0; i < len(array); i++ {
		if array[i] == val {
			index = i
			return index
		}
	}
	return index
}
func QueryUsersOfWorkerHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	var user WorkerUsers

	vars := mux.Vars(req)
	if workername, isFound := vars["name"]; isFound {
		user.Workername = workername

		Info.Println("QueryUsersOfWorkerHandler workername:", workername)

		gPool.minersLock.RLock()
		for miner, _ := range gPool.miners {
			if miner.workername == workername {
				if StringsContains(user.Users,miner.username) == -1 {
					user.Users = append(user.Users, miner.username)
					user.UserCount++
				}
			}
		}
		gPool.minersLock.RUnlock()

		userJson, err := json.Marshal(&user)
		if err != nil {
			Warning.Println("userJson Marshal error:", err.Error())
		}

		userStr := string(userJson) + "\n"
		n, err := res.Write([]byte(userStr))
		if err != nil {
			Warning.Println("userJson Write error:", err.Error())
		} else if n != len(userStr) {
			Warning.Printf("userJson Write %d!=%d", n, len(userStr))
		}
	}
}

type User struct {
	UserCount int      `json:"userCount"`
	Users     []string `json:"users"`
}

func QueryUserHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	var user User

	userMap := make(map[string]int)

	Info.Println("QueryUserHandler:")

	gPool.minersLock.RLock()
	for miner, _ := range gPool.miners {
		if _, isFound := userMap[miner.username]; !isFound {
			userMap[miner.username] = 1
			Info.Println(miner.username)
		}
	}
	gPool.minersLock.RUnlock()

	for username, _ := range userMap {
		user.Users = append(user.Users, username)
		user.UserCount++
	}

	userJson, err := json.Marshal(&user)
	if err != nil {
		Warning.Println("userJson Marshal error:", err)
	}

	userStr := string(userJson) + "\n"
	n, err := res.Write([]byte(userStr))
	if err != nil {
		Warning.Println("userJson Write error:", err)
	} else if n != len(userStr) {
		Warning.Printf("userJson Write %d!=%d", n, len(userStr))
	}
}

type TotalInfo struct {
	TotalPoolPower   string `json:"totalpoolpower"`
	TotalPower    string `json:"totalpower"`
	TotalUsers    int    `json:"totalusers"`
	TotalWorker   int    `json:"totalworkers"`
	CurrentHeight int64  `json:"currentheight"`
	CurrentDiff   string `json:"currentdiff"`
	TotalRewards  string `json:"totalrewards"`
	TotalSentRewards string `json:"totalsentrewards"`
}

func QueryTotalInfoHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	var totalinfo TotalInfo

	userMap := make(map[string]int)

	Info.Println("QueryTotalInfoHandler:")

	gPool.minersLock.RLock()
	var totalpoolpower, totalpower float64
	for miner, _ := range gPool.miners {
		if _, isFound := userMap[miner.username]; !isFound {
			userMap[miner.username] = 1
			Info.Println(miner.username)
		}
		totalinfo.TotalWorker++
		totalpoolpower += float64(miner.acceptShare) / 1000 / 1000 / time.Since(miner.onlineTime).Seconds() * 4 * 1024 * 1024 * 1024
	}

	session, err := mgo.Dial(gPool.cfg.MongoDB.Url)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	Info.Println("QueryBlocksInfoHandlerHandler:")

	foundblocks_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.BlockCol)
	blockcount, _ := foundblocks_t.Find(nil).Count()

	var senttxs []SendTx_t
	var totalsentrewards int64
	sendtxs_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.SendTx)
	sendtxs_t.Find(bson.M{"txstate": 3}).All(&senttxs)
	if len(senttxs) == 0 {
		totalinfo.TotalSentRewards = "0"
	} else {
		for _, item := range senttxs {
			totalsentrewards += item.Amount
		}
	}

	gPool.minersLock.RUnlock()
	totalinfo.TotalPoolPower = fmt.Sprintf("%f MHash/s", totalpoolpower)
	totalinfo.TotalUsers = len(userMap)

	currdiff, _ := strconv.ParseFloat(gPool.LastJob.netDiffStr, 64)
	totalinfo.CurrentDiff = fmt.Sprintf("%f", float64(currdiff*256))
	totalpower = math.Pow(2, 24) * currdiff / 60 / 1024 / 1024 * 256
	totalinfo.TotalPower = fmt.Sprintf("%f MHash/s", totalpower)
	totalinfo.CurrentHeight = gPool.height - 1
	totalinfo.TotalRewards = fmt.Sprintf("%.02f", float64(int64(blockcount)*cfg.OneBlockReward/100000000))
	totalinfo.TotalSentRewards = fmt.Sprintf("%.02f", float64(int64(totalsentrewards)/100000000))

	userJson, err := json.Marshal(&totalinfo)
	if err != nil {
		Warning.Println("totalinfoJson Marshal error:", err)
	}

	userStr := string(userJson) + "\n"
	n, err := res.Write([]byte(userStr))
	if err != nil {
		Warning.Println("totalinfoJson Write error:", err)
	} else if n != len(userStr) {
		Warning.Printf("totalinfoJson Write %d!=%d", n, len(userStr))
	}
}

type Rewards struct {
	TotalRewards int64 `json:"totalrewards"`
	SentRewards  int64 `json:"sentrewards"`
}

func QueryRewardsHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	vars := mux.Vars(req)
	if username, isFound := vars["name"]; isFound {
		username = strings.ToLower(username)
		Info.Println("QueryRewardsHandler username:", username)

		session, err := mgo.Dial(gPool.cfg.MongoDB.Url)
		if err != nil {
			panic(err)
		}
		defer session.Close()

		var rewards Rewards

		Info.Println("QueryRewardsHandler:")

		var minerinfo []MinerInfo_t
		minerinfo_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.MinerInfo)
		minerinfo_t.Find(bson.M{"uname": username}).All(&minerinfo)

		var totalRewards int64
		for _, item := range minerinfo {
			totalRewards += item.Reward
		}

		var senttx []SendTx_t
		senttx_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.SendTx)
		senttx_t.Find(bson.M{"uname": username}).All(&senttx)

		var sentRewards int64
		for _, item := range senttx {
			sentRewards += item.Amount
		}

		rewards.TotalRewards = totalRewards
		rewards.SentRewards = sentRewards

		rewardsJson, err := json.Marshal(&rewards)
		if err != nil {
			Warning.Println("rewardsJson Marshal error:", err)
		}

		rewardsStr := string(rewardsJson) + "\n"
		n, err := res.Write([]byte(rewardsStr))
		if err != nil {
			Warning.Println("userJson Write error:", err)
		} else if n != len(rewardsStr) {
			Warning.Printf("userJson Write %d!=%d", n, len(rewardsStr))
		}
	}
}

type Share struct {
	OnlineTime string  `json:"onlineTime"`
	Accept     int     `json:"accept"`
	Reject     int     `json:"reject"`
	Stale      int     `json:"stale"`
	Duplicate  int     `json:"duplicate"`
	Diff       float64 `json:"diff"`
	HashRate   string  `json:"hashRate"`
}

func QueryShareHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	var username, workername string

	vars := mux.Vars(req)
	if name, isFound := vars["name"]; isFound {
		names := strings.SplitN(name, ".", 2)
		if len(names) == 2 {
			username = names[0]
			username = strings.ToLower(username)
			workername = names[1]
		} else {
			return
		}

		var share Share

		gPool.minersLock.RLock()
		for miner, _ := range gPool.miners {
			if miner.username == username && miner.workername == workername {
				share.OnlineTime = miner.onlineTime.Format("2006-01-02 15:04:05")
				share.Accept = miner.acceptNum
				share.Reject = miner.rejectNum
				share.Stale = miner.staleNum
				share.Duplicate = miner.duplicateNum
				share.Diff = miner.diff
				share.HashRate = fmt.Sprintf("%fM/s", miner.acceptShare/1000/1000/time.Since(miner.onlineTime).Seconds()*4*1024*1024*4)
				break
			}
		}
		gPool.minersLock.RUnlock()

		shareJson, err := json.Marshal(&share)
		if err != nil {
			Warning.Println("QueryShareHandler Marshal error:", err)
		}

		shareStr := string(shareJson) + "\n"
		n, err := res.Write([]byte(shareStr))
		if err != nil {
			Warning.Println("QueryShareHandler Write error:", err)
		} else if n != len(shareStr) {
			Warning.Printf("QueryShareHandler Write %d!=%d", n, len(shareStr))
		}
	}
}

type OneBlock struct {
	BlockHeight int64  `json:"blockheight"`
	Miner       string `json:"miner"`
	BlockReward string `json:"blockreward"`
	BlockTime   int64  `json:"blocktime"`
	BlockHash   string `json:"blockhash"`
}

func QueryBlocksInfoHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	session, err := mgo.Dial(gPool.cfg.MongoDB.Url)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	Info.Println("QueryBlocksInfoHandlerHandler:")

	var foundblocks []FoundBlock_t
	foundblocks_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.BlockCol)
	foundblocks_t.Find(nil).Sort("-stime").Limit(10).All(&foundblocks)

	blocksinfo := make([]OneBlock, 10)
	for k, item := range foundblocks {
		var oneblock OneBlock
		reverseStr, _ := reverseS(item.BlockHash)
		oneblock.BlockHash = reverseStr
		oneblock.BlockHeight = item.Number
		oneblock.BlockReward = fmt.Sprintf("%.02f", float64(cfg.OneBlockReward/100000000))
		oneblock.BlockTime = time.Now().Unix() - item.ShareTime
		oneblock.Miner = item.Worker
		blocksinfo[k] = oneblock
	}

	blocksinfoJson, err := json.Marshal(&blocksinfo)
	if err != nil {
		Warning.Println("blocksinfoJson Marshal error:", err)
	}

	blocksinfoStr := string(blocksinfoJson) + "\n"
	n, err := res.Write([]byte(blocksinfoStr))
	if err != nil {
		Warning.Println("userJson Write error:", err)
	} else if n != len(blocksinfoStr) {
		Warning.Printf("userJson Write %d!=%d", n, len(blocksinfoStr))
	}
}

type WorkerReward struct {
	TotalRewards int64  `json:"value"`
	WorkerName   string `json:"name"`
}

func QueryRewardsInfoHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	Info.Println("QueryRewardsInfoHandler.")

	session, err := mgo.Dial(gPool.cfg.MongoDB.Url)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	var minerinfo []MinerInfo_t
	minerinfo_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.MinerInfo)
	minerinfo_t.Find(bson.M{"stattime": bson.M{"$gte": time.Now().Unix() - 24*3600, "$lt": time.Now().Unix()}}).All(&minerinfo)

	workermap := make(map[string]int64)
	for _, item := range minerinfo {
		if item.Reward == 0 && item.Worker == "" {
			continue
		}
		if _, isFound := workermap[item.Worker]; isFound {
			workermap[item.Worker] += item.Reward
		} else {
			workermap[item.Worker] = item.Reward
		}
	}

	workerrewards := make([]WorkerReward, len(workermap))
	var temp int
	temp = 0
	for k, v := range workermap {
		var oneworker WorkerReward
		oneworker.TotalRewards = v
		oneworker.WorkerName = k
		workerrewards[temp] = oneworker
		temp++
	}

	workerrewardsinfoJson, err := json.Marshal(&workerrewards)
	if err != nil {
		Warning.Println("workerrewardsinfoJson Marshal error:", err)
	}

	workerrewardsinfoStr := string(workerrewardsinfoJson) + "\n"
	n, err := res.Write([]byte(workerrewardsinfoStr))
	if err != nil {
		Warning.Println("userJson Write error:", err)
	} else if n != len(workerrewardsinfoStr) {
		Warning.Printf("userJson Write %d!=%d", n, len(workerrewardsinfoStr))
	}
}

type WorkerInfo struct {
	AvgPower      string `json:"avgpower"`
	WorkerCount   int64  `json:"workercount"`
	ValidShares   int64  `json:"validshares"`
	InvalidShares int64  `json:"invalidshares"`
	TotalRewards  string `json:"totalrewards"`
	SentRewards   string `json:"sentrewards"`
}

func QueryMinerInfoHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")
	vars := mux.Vars(req)

	timeStr := time.Now().Format("2006-01-02")
	t, _ := time.ParseInLocation("2006-01-02", timeStr, time.Local)
	timeNumber := t.Unix()

	var username, workername string
	queryFlag := false
	if name, isFound := vars["name"]; isFound {
		if strings.Contains(name, ".") == true {
			names := strings.SplitN(name, ".", 2)
			if len(names) == 2 {
				username = names[0]
				username = strings.ToLower(username)
				workername = names[1]
			} else {
				return
			}
		} else {
			queryFlag = true
			username = name
			workername = name
		}
		Info.Println("QueryMinerInfoHandler username:", username)

		session, err := mgo.Dial(gPool.cfg.MongoDB.Url)
		if err != nil {
			panic(err)
		}
		defer session.Close()

		var workerinfo WorkerInfo

		var minerinfo []MinerInfo_t
		minerinfo_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.MinerInfo)
		if queryFlag == true {
			minerinfo_t.Find(bson.M{"uname": username}).All(&minerinfo)
			if len(minerinfo) == 0 {
				minerinfo_t.Find(bson.M{"worker": workername}).All(&minerinfo)
			}
		} else {
			minerinfo_t.Find(bson.M{"uname": username, "worker": workername}).All(&minerinfo)
		}

		var totalRewards int64
		for _, item := range minerinfo {
			totalRewards += item.Reward
		}

		var senttx []SendTx_t
		senttx_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.SendTx)
		if queryFlag == true {
			senttx_t.Find(bson.M{"uname": username, "txstate": 3}).All(&senttx)
			if len(senttx) == 0 {
				senttx_t.Find(bson.M{"worker": workername, "txstate": 3}).All(&senttx)
			}
		} else {
			senttx_t.Find(bson.M{"uname": username, "worker": workername, "txstate": 3}).All(&senttx)
		}

		var sentRewards int64
		for _, item := range senttx {
			sentRewards += item.Amount
		}

		shares_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.ShareCol)
		var totalonedayvalidshares []Share_t
		var totalvalidshares, totalinvalidshares, totalonedayshares int
		if queryFlag == true {
			totalvalidshares, _ = shares_t.Find(bson.M{"uname": username, "valid": true}).Count()
			totalinvalidshares, _ = shares_t.Find(bson.M{"uname": username, "valid": false}).Count()
			shares_t.Find(bson.M{"uname": username, "valid": true, "stime": bson.M{"$gte": timeNumber - 24*3600, "$lt": timeNumber}}).All(&totalonedayvalidshares)
			totalonedayshares = len(totalonedayvalidshares)
			if totalvalidshares == 0 && totalinvalidshares == 0 && totalonedayshares == 0 {
				totalvalidshares, _ = shares_t.Find(bson.M{"valid": true, "worker": workername}).Count()
				totalinvalidshares, _ = shares_t.Find(bson.M{"worker": workername, "valid": false}).Count()
				shares_t.Find(bson.M{"worker": workername, "valid": true, "stime": bson.M{"$gte": timeNumber - 24*3600, "$lt": timeNumber}}).All(&totalonedayvalidshares)
				totalonedayshares = len(totalonedayvalidshares)
			}
		} else {
			totalvalidshares, _ = shares_t.Find(bson.M{"uname": username, "valid": true, "worker": workername}).Count()
			totalinvalidshares, _ = shares_t.Find(bson.M{"uname": username, "worker": workername, "valid": false}).Count()
			shares_t.Find(bson.M{"uname": username, "valid": true, "worker": workername, "stime": bson.M{"$gte": timeNumber - 24*3600, "$lt": timeNumber}}).All(&totalonedayvalidshares)
			totalonedayshares = len(totalonedayvalidshares)
		}
		var totaldiff float64
		for _, item := range totalonedayvalidshares {
			diff, _ := strconv.ParseFloat(item.Share, 64)
			totaldiff += diff
		}
		workerinfo.AvgPower = fmt.Sprintf("%.03fM/s", totaldiff/1000/1000/24/3600*4*1024*1024*1024)
		gPool.minersLock.RLock()
		for miner, _ := range gPool.miners {
			if queryFlag == true {
				if miner.username == username || miner.workername == workername {
					workerinfo.WorkerCount++
				}
			} else {
				if miner.username == username && miner.workername == workername {
					workerinfo.WorkerCount++
				}
			}
		}
		gPool.minersLock.RUnlock()

		workerinfo.ValidShares = int64(totalvalidshares)
		workerinfo.InvalidShares = int64(totalinvalidshares)
		totalRewards = totalRewards * (100 - cfg.PoolFeeRate) / 100
		workerinfo.TotalRewards = fmt.Sprintf("%.08f", float64(totalRewards)/100000000)
		workerinfo.SentRewards = fmt.Sprintf("%.08f", float64(sentRewards)/100000000)

		workerinfoJson, err := json.Marshal(&workerinfo)
		if err != nil {
			Warning.Println("workerinfoJson Marshal error:", err)
		}

		workerinfoStr := string(workerinfoJson) + "\n"
		n, err := res.Write([]byte(workerinfoStr))
		if err != nil {
			Warning.Println("workerinfoJson Write error:", err)
		} else if n != len(workerinfoStr) {
			Warning.Printf("workerinfoJson Write %d!=%d", n, len(workerinfoStr))
		}
	}
}

type WorkerHanfHourInfo struct {
	TimeStamp int64  `json:"timestamp"`
	Power     string `json:"power"`
	Rewards   string `json:"rewards"`
}

func QueryWorkerHanfHourInfoHandler(res http.ResponseWriter, req *http.Request) {

	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	timeNumber := time.Now().Unix() / 1800 * 1800
	queryFlag := false

	vars := mux.Vars(req)
	if name, isFound := vars["name"]; isFound {
		var username, workername string
		if strings.Contains(name, ".") == true {
			names := strings.SplitN(name, ".", 2)
			if len(names) == 2 {
				username = names[0]
				username = strings.ToLower(username)
				workername = names[1]
			} else {
				return
			}
		} else {
			queryFlag = true
			username = name
			workername = name
		}
		session, err := mgo.Dial(gPool.cfg.MongoDB.Url)
		if err != nil {
			panic(err)
		}
		defer session.Close()

		Info.Println("QueryWorkerHanfHourInfoHandler:")

		workhalfinfos := make([]WorkerHanfHourInfo, 48)

		for i := 0; i < 48; i++ {
			var minerinfo []MinerInfo_t
			minerinfo_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.MinerInfo)
			if queryFlag == true {
				minerinfo_t.Find(bson.M{"uname": username, "stattime": bson.M{"$gte": timeNumber - 1800, "$lt": timeNumber}}).All(&minerinfo)
				if len(minerinfo) == 0 {
					minerinfo_t.Find(bson.M{"worker": workername, "stattime": bson.M{"$gte": timeNumber - 1800, "$lt": timeNumber}}).All(&minerinfo)
				}
			} else {
				minerinfo_t.Find(bson.M{"uname": username, "worker": workername, "stattime": bson.M{"$gte": timeNumber - 1800, "$lt": timeNumber}}).All(&minerinfo)
			}
			var totalRewards int64
			for _, item := range minerinfo {
				totalRewards += item.Reward
			}

			shares_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.ShareCol)
			var shares []Share_t
			if queryFlag == true {
				shares_t.Find(bson.M{"uname": username, "valid": true, "stime": bson.M{"$gte": timeNumber - 1800, "$lt": timeNumber}}).All(&shares)
				if len(shares) == 0 {
					shares_t.Find(bson.M{"worker": workername, "valid": true, "stime": bson.M{"$gte": timeNumber - 1800, "$lt": timeNumber}}).All(&shares)
				}
			} else {
				shares_t.Find(bson.M{"uname": username, "worker": workername, "valid": true, "stime": bson.M{"$gte": timeNumber - 1800, "$lt": timeNumber}}).All(&shares)
			}
			var totaldiff float64
			for _, item := range shares {
				diff, _ := strconv.ParseFloat(item.Share, 64)
				totaldiff += diff
			}
			power := fmt.Sprintf("%.03f", totaldiff/1000/1000/1800*4*1024*1024*1024)

			var workhalfinfo WorkerHanfHourInfo

			workhalfinfo.Rewards = fmt.Sprintf("%.08f", float64(totalRewards)/100000000)
			workhalfinfo.Power = power
			workhalfinfo.TimeStamp = timeNumber

			workhalfinfos[i] = workhalfinfo
			timeNumber -= 1800
		}

		workhalfinfosJson, err := json.Marshal(&workhalfinfos)
		if err != nil {
			Warning.Println("workhalfinfosJson Marshal error:", err)
		}

		workhalfinfosStr := string(workhalfinfosJson) + "\n"
		n, err := res.Write([]byte(workhalfinfosStr))
		if err != nil {
			Warning.Println("workhalfinfosJson Write error:", err)
		} else if n != len(workhalfinfosStr) {
			Warning.Printf("workhalfinfosJson Write %d!=%d", n, len(workhalfinfosStr))
		}
	}
}

type PowerDiffInfo struct {
	TimeStamp int64  `json:"timestamp"`
	Power     string `json:"power"`
	Diff      string `json:"diff"`
}

func QueryPowerDiffInfoHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	res.Header().Set("content-type", "application/json")

	timeNumber := time.Now().Unix() / 3600 * 3600

	session, err := mgo.Dial(gPool.cfg.MongoDB.Url)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	Info.Println("QueryPowerDiffInfoHandler:")

	powerdiffinfos := make([]PowerDiffInfo, 24)

	for i := 0; i < 24; i++ {
		var shares []Share_t
		shares_t := session.DB(gPool.cfg.MongoDB.DBname).C(gPool.cfg.MongoDB.ShareCol)
		shares_t.Find(bson.M{"stime": bson.M{"$gte": timeNumber - 3600, "$lt": timeNumber}}).All(&shares)
		totalshares, _ := shares_t.Find(bson.M{"stime": bson.M{"$gte": timeNumber - 3600, "$lt": timeNumber}}).Count()
		//power := fmt.Sprintf("%.02f", float64(totalshares)/1000/1000/3600*4*1024*1024*1024)
		var totaldiff, totalsharediff float64
		totaldiff = 0.00
		totalsharediff = 0.00
		for _, item := range shares {
			diff, _ := strconv.ParseFloat(item.NetDiff, 64)
			sharediff, _ := strconv.ParseFloat(item.Share, 64)
			totaldiff += diff
			totalsharediff += sharediff
		}

		var powerdiffinfo PowerDiffInfo

		powerdiffinfo.Diff = fmt.Sprintf("%f", float64(totaldiff*256)/float64(totalshares))
		power := fmt.Sprintf("%.02f", float64(totalsharediff)/1000/1000/3600*4*1024*1024*1024)
		//power := math.Pow(2, 24) * float64(totaldiff) / float64(totalshares) / 60 / 1024 / 1024 * 256
		powerdiffinfo.Power = power
		powerdiffinfo.TimeStamp = timeNumber

		powerdiffinfos[i] = powerdiffinfo
		timeNumber -= 3600
	}

	powerdiffinfosJson, err := json.Marshal(&powerdiffinfos)
	if err != nil {
		Warning.Println("powerdiffinfosJson Marshal error:", err)
	}

	powerdiffinfosStr := string(powerdiffinfosJson) + "\n"
	n, err := res.Write([]byte(powerdiffinfosStr))
	if err != nil {
		Warning.Println("powerdiffinfosJson Write error:", err)
	} else if n != len(powerdiffinfosStr) {
		Warning.Printf("powerdiffinfosJson Write %d!=%d", n, len(powerdiffinfosStr))
	}
}

func (pool *Pool) httpServer() {

	r := mux.NewRouter()
	r.HandleFunc("/share/{name}", QueryShareHandler)
	r.HandleFunc("/ip/{name}", QueryIPHandler)
	r.HandleFunc("/worker/{name}", QueryWorkerHandler)
	r.HandleFunc("/users/{name}", QueryUsersOfWorkerHandler)
	r.HandleFunc("/rewards/{name}", QueryRewardsHandler)
	r.HandleFunc("/user", QueryUserHandler)
	r.HandleFunc("/totalinfo", QueryTotalInfoHandler)
	r.HandleFunc("/blocksinfo", QueryBlocksInfoHandler)
	r.HandleFunc("/rewardsinfo", QueryRewardsInfoHandler)
	r.HandleFunc("/powerdiffinfo", QueryPowerDiffInfoHandler)
	r.HandleFunc("/minerinfo/{name}", QueryMinerInfoHandler)
	r.HandleFunc("/minerrewardinfo/{name}", QueryWorkerHanfHourInfoHandler)
	http.Handle("/", r)

	httpServer := &http.Server{
		Addr:           pool.cfg.HttpIPPort,
		Handler:        r,
		MaxHeaderBytes: pool.cfg.LimitHeadersSize,
	}
	Info.Printf("httpServer listening on %s", pool.cfg.HttpIPPort)
	err := httpServer.ListenAndServe()
	if err != nil {
		Alert.Fatalln("ListenAndServe error: ", err)
	}
}
