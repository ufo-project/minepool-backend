package main

import (
	"github.com/jasonlvhit/gocron"
	"strconv"
	"strings"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func (pool *Pool) openLevelDB() {
	var err error
	var i int

	if pool.levelDB != nil {
		pool.levelDB.Close()
		pool.levelDB = nil
	}

	for i = 0; i < 3; i++ {
		pool.levelDB, err = leveldb.OpenFile(pool.cfg.LevelDB, nil)
		if err != nil {
			if pool.levelDB != nil {
				pool.levelDB.Close()
				pool.levelDB = nil
			}
			Info.Println("Unable to open leveldb connection:", err)
			time.Sleep(time.Second * 1)
		} else {
			break
		}
	}

	if i == 3 {
		Alert.Panic("openLevelDB 3 times failed.")
	}
	Info.Println("openLevelDB:", pool.cfg.LevelDB, i+1, " times success.")
}

func (pool *Pool) OpenMongoDB() bool {
	var err error
	pool.mgoSession, err = mgo.Dial(pool.cfg.MongoDB.Url)
	if err != nil {
		Warning.Println("Unable to connect mgdb", gPool.cfg.MongoDB.Url, "error:", err)
		return false
	}

	pool.mgoSession.SetMode(mgo.Monotonic, true)
	pool.mgoSession.SetPoolLimit(5)
	pool.mgoSession.SetSocketTimeout(time.Second * 5)
	pool.mgoSession.SetSyncTimeout(time.Second * 5)
	Info.Println("OpenMongoDB success")
	return true
}

func (pool *Pool) GetSession() *mgo.Session {
	if pool.mgoSession == nil {
		if !pool.OpenMongoDB() {
			Alert.Fatalln("GetSession(): OpenMongoDB error")
		}
	}
	return pool.mgoSession.Clone()
}

func (pool *Pool) closeDB() {
	pool.mgoSession.Close()
	pool.levelDB.Close()
}

type Share_t struct {
	Id        bson.ObjectId `bson:"_id"`
	Username  string        `bson:"uname"`
	Worker    string        `bson:"worker"`
	Share     string        `bson:"sdiff"`
	Valid     bool          `bson:"valid"`
	NetDiff   string        `bson:"ndiff"`
	ShareTime int64         `bson:"stime"`
}

type FoundBlock_t struct {
	Id        bson.ObjectId `bson:"_id"`
	Username  string        `bson:"uname"`
	Worker    string        `bson:"worker"`
	Number    int64         `bson:"number"`
	ShareTime int64         `bson:"stime"`
	BlockHash string        `bson:"blockhash"`
}

type MinerInfo_t struct {
	Id       bson.ObjectId `bson:"_id"`
	Username string        `bson:"uname"`
	Reward   int64         `bson:"reward"`
	StatTime int64         `bson:"stattime"`
	Worker   string        `bson:"worker"`
}

type AddrBalance_t struct {
	Id       bson.ObjectId `bson:"_id"`
	Username string        `bson:"uname"`
	Balance   int64         `bson:"balance"`
	UpdateTime int64         `bson:"updatetime"`
	Worker   string        `bson:"worker"`
}

type SendTx_t struct {
	Id       bson.ObjectId `bson:"_id"`
	TxId     string        `bson:"txid"`
	Username string        `bson:"uname"`
	Worker   string        `bson:"worker"`
	Amount   int64         `bson:"amount"`
	SendTime int64         `bson:"sendtime"`
	TxState  int64         `bson:"txstate"` //Pending(0), InProgress(1), Cancelled(2), Completed(3), Failed(4), Registering(5)
	ResendFlag int64         `bson:"resendflag"`
}

func (miner *Miner) insertShareToLevelDB(key, valid, netDiff string) {
	sharetime := time.Now().Unix()

	value := miner.username + `|` + miner.workername + `|` + miner.diffStr + `|` + valid + `|` + netDiff + `|` + strconv.FormatInt(sharetime, 10)

	err := miner.pool.levelDB.Put([]byte(key), []byte(value), nil)
	if err != nil {
		Warning.Println("insert leveldb error:", err)
		miner.pool.openLevelDB()
	}
}

func (pool *Pool) pushShares(session *mgo.Session) {
	var count int
	share_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.ShareCol)
	bulk := share_t.Bulk()

	iter := pool.levelDB.NewIterator(nil, nil)
	defer func() {
		iter.Release()
		err := iter.Error()
		if err != nil {
			Warning.Println("pushShares iter shares error:", err)
		}
	}()

	start := time.Now()
	for iter.Next() {

		key := iter.Key()
		value := iter.Value()

		str := strings.Split(string(value), "|")
		if len(str) != 6 {
			continue
		}

		username := str[0]
		workername := str[1]
		share := str[2]

		var valid bool
		if str[3] == "1" {
			valid = true
		} else {
			valid = false
		}

		netdiff := str[4]

		sharetime, err := strconv.ParseInt(str[5], 10, 64)
		if err != nil {
			Warning.Println("pushShares parse time error:", err)
			sharetime = time.Now().Unix()
		}

		Info.Println("pushShares Share_t username:", username, "workname:", workername, "sharetime:", sharetime)
		shareMdb := &Share_t{Id: bson.NewObjectId(), Username: username, Worker: workername, Share: share, Valid: valid, NetDiff: netdiff, ShareTime: sharetime}
		bulk.Insert(shareMdb)

		err = pool.levelDB.Delete([]byte(key), nil)
		if err != nil {
			pool.openLevelDB()
		}

		count++
		if count == 100 {
			count = 0
			_, err := bulk.Run()
			if err != nil {
				Warning.Println("bulk run error:", err)
			}
		}
	}
	if count > 0 {
		_, err := bulk.Run()
		if err != nil {
			Warning.Println("bulk run error:", err)
		}
	}

	Info.Println("pushShares", count, "finished:", time.Since(start))
	Info.Println("pool stale:", pool.staleCount, "duplicate:", pool.duplicateCount, "lowDiff:", pool.lowDiffCount, "accept:", pool.acceptCount)
}

func (pool *Pool) pushFoundBlock(session *mgo.Session) {
	Info.Println("pushFoundBlock start.")
	pool.found_block_mapLock.Lock()
	defer pool.found_block_mapLock.Unlock()

	foundblock_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.BlockCol)

	start := time.Now()
	for k, v := range pool.found_block_map {
		str := strings.Split(string(v), "|")
		if len(str) == 5 {
			height, err := strconv.ParseInt(str[2], 10, 64)
			if err != nil {
				Warning.Println("parse height error:", err)
			}
			time, err := strconv.ParseInt(str[3], 10, 64)
			if err != nil {
				Warning.Println("parse share foundtime string to int failed", err)
			}
			block := &FoundBlock_t{Id: bson.NewObjectId(), Username: str[0], Worker: str[1], Number: height, ShareTime: time, BlockHash: str[4]}
			err = foundblock_t.Insert(block)
			if err != nil {
				Warning.Println("writemdb foundblock error:", err)
				return
			}
			Info.Println("pushFoundBlock insert:", height, "blocktime:", time)
		}
		delete(pool.found_block_map, k)
	}

	Info.Println("pushFoundBlock finished:", time.Since(start))
}

func (pool *Pool) isEmptyMinerInfo(session *mgo.Session) int64 {
	minerinfo_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.MinerInfo)
	totalnum, err := minerinfo_t.Count()
	timenow := time.Now().Unix()
	if err != nil {
		Warning.Println("isEmptyMinerInfo:minerinfo_t collection get count error.:", err)
		return -1
	}

	if totalnum == 0 {
		Info.Println("isEmptyMinerInfo: minerinfo_t is empty,insert init_user.")
		minerinfo := &MinerInfo_t{Id: bson.NewObjectId(), Username: "init_user", Reward: 0, StatTime: timenow, Worker: "init_worker"}
		err := minerinfo_t.Insert(minerinfo)
		if err != nil {
			Warning.Println("writemdb minerinfo error:", err)
			return -1
		}
		return 1
	} else if totalnum > 0 {
		var item MinerInfo_t
		minerinfo_t.Find(nil).Sort("-stattime").One(&item)
		return item.StatTime
	}

	return 0
}

func (pool *Pool) UpdateAddrBalance(session *mgo.Session) int64 {
	Info.Println("UpdateAddrBalance start.")
	pool.minersLock.RLock()
	defer pool.minersLock.RUnlock()

	nTime := time.Now()
	yesTime := nTime.AddDate(0, 0, -1)
	yesTimeStr := (yesTime.Format("2006-01-02") + " 00:00:00")
	formatTimeyes, _ := time.ParseInLocation("2006-01-02 15:04:05", yesTimeStr, time.Local)
	nTimeStr := nTime.Format("2006-01-02") + " 00:00:00"
	formatTime, _ := time.ParseInLocation("2006-01-02 15:04:05", nTimeStr, time.Local)
	minerinfo_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.MinerInfo)

	var minerinfo []MinerInfo_t
	err := minerinfo_t.Find(bson.M{"stattime": bson.M{"$gte": formatTimeyes.Unix(), "$lt": formatTime.Unix()}}).All(&minerinfo)
	if err != nil {
		Warning.Println("UpdateAddrBalance:minerinfo_t collection get count error:", err.Error())
		return -1
	}

	if len(minerinfo) == 0 {
		Warning.Println("UpdateAddrBalance:minerinfo_t collection is empty,do not need send reward.")
		return 0
	} else if len(minerinfo) > 0 {
		var miners_map map[string]int64
		miners_map = make(map[string]int64)
		for _, item := range minerinfo {
			_, ok := miners_map[item.Username+"."+item.Worker]
			if item.Username == "" && item.Worker == "" {
				continue
			}
			if ok {
				miners_map[item.Username+"."+item.Worker] += item.Reward
			} else {
				miners_map[item.Username+"."+item.Worker] = item.Reward
			}
		}

		addrbalance_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.AddrBalance)
		for k, v := range miners_map {
			names := strings.SplitN(k, ".", 2)
			if len(names) == 2 {
				username := names[0]
				username = strings.ToLower(username)
				workername := names[1]
				if v <= 0 {
					continue
				}
				v = v * (100 - cfg.PoolFeeRate) / 100

				var item []AddrBalance_t
				err := addrbalance_t.Find((bson.M{"uname": username})).All(&item)
				if err != nil {
					Warning.Println("UpdateAddrBalance:addrbalance_t.Insert usernanme:", username, ",workname:", workername, ",amount:", v, "error:", err.Error())
					continue
				}

				if len(item) == 0 {
					oneaddrbalance := &AddrBalance_t{Id: bson.NewObjectId(), Username: username, Balance : v, UpdateTime: time.Now().Unix(), Worker: workername}
					err = addrbalance_t.Insert(oneaddrbalance)
					if err != nil {
						Warning.Println("UpdateAddrBalance:addrbalance_t collection insert error:", err.Error())
						continue
					}
				} else if len(item) == 1 {
					data := bson.M{"$set": bson.M{"balance": item[0].Balance+v,"updatetime":time.Now().Unix()}}
					err := addrbalance_t.UpdateId(item[0].Id, data)
					if err != nil {
						Warning.Println("UpdateAddrBalance:addrbalance_t.Update balance:", item[0].Balance+v, "err:", err.Error())
					}
				}
			}
		}
	}

	return 0
}

func (pool *Pool) SendReward(session *mgo.Session) int64 {
	Info.Println("SendReward start.")

	var balances []AddrBalance_t
	addrbalance_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.AddrBalance)
	err := addrbalance_t.Find(bson.M{"balance": bson.M{"$gte": cfg.SendMinUfo}}).All(&balances)
	if err != nil {
		Warning.Println("SendReward:addrbalance_t collection get all error:", err.Error())
		return -1
	}
	if len(balances) == 0 {
		Warning.Println("SendReward:addrbalance_t collection have no record balance > 5ufo,do not need send reward.")
		return 0
	} else if len(balances) > 0 {
		sendtx_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.SendTx)
		var i int
		for i=0; i< len(balances); {
			walletbalance,_ := getWalletStatus()
			var ten_item_balance int64
			ten_item_balance = 0
			endindex := 0
			if i+10 > len(balances) {
				endindex = len(balances)
			} else {
				endindex = i + 10
			}
			for j:=i;j<endindex;j++ {
				ten_item_balance += balances[j].Balance
			}
			if walletbalance < ten_item_balance {
				Alert.Println("Wallet have no enough balance.")
				return 0
			}

			var txids []string
			for k:=i;k<endindex;k++ {
				txid, err := sendTx(balances[k].Balance, balances[k].Username)
				if err != nil {
					Warning.Println("sendReward:sendTx to:", balances[k].Username, ",amount:", balances[k].Balance, "error:", err.Error())
					continue
				}
				data := bson.M{"$set": bson.M{"balance": 0, "updatetime": time.Now().Unix()}}
				err = addrbalance_t.UpdateId(balances[k].Id, data)
				if err != nil {
					Warning.Println("sendReward:addrbalance_t.Update balance:", 0, "err:", err.Error())
				}
				sendonetx := &SendTx_t{Id: bson.NewObjectId(), TxId: txid, Username: balances[k].Username, Amount: balances[k].Balance, SendTime: time.Now().Unix(), TxState: 0, Worker: balances[k].Worker, ResendFlag: 0}
				err = sendtx_t.Insert(sendonetx)
				Info.Println("sendReward:sendtx_t. before Insert usernanme:", balances[k].Username, ",workname:", balances[k].Worker, ",amount:", balances[k].Balance)
				if err != nil {
					Warning.Println("sendReward:sendtx_t.Insert usernanme:", balances[k].Username, ",workname:", balances[k].Worker, ",amount:", balances[k].Balance, "error:", err.Error())
					continue
				}
				txids = append(txids, txid)
			}

			time.Sleep(time.Minute * 10)

			for _,item := range txids {
				txstate,err := getTxState(item)
				if err != nil {
					Warning.Println("sendReward:getTxState. txid:", item, "error:", err.Error())
					continue
				}
				if txstate == 1 {
					success,err := cancelTx(item)
					if err != nil {
						Warning.Println("sendReward:cancelTx txid:", item, "error:", err.Error())
						continue
					}
					if success {
						selector := bson.M{"txid": item}
						data := bson.M{"$set": bson.M{"txstate": 4}}
						err := sendtx_t.Update(selector, data)
						if err != nil {
							Warning.Println("CheckTxs:sendtx_t.Update:", 4, "err:", err)
							return -1
						}
					}
				}
			}

			i += 10
		}
	}

	return 0
}

func (pool *Pool) ReSendTxs(session *mgo.Session) int64 {
	Info.Println("ReSendTxs start.")

	var failtxs []SendTx_t
	sendtx_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.SendTx)
	err := sendtx_t.Find(bson.M{"txstate": 4}).All(&failtxs)
	if err != nil {
		Warning.Println("ReSendTxs:sendtx_t collection get failed tx error:", err.Error())
		return -1
	}
	if len(failtxs) == 0 {
		Warning.Println("ReSendTxs:sendtx_t collection have no failed tx,do not need resend reward.")
		return 0
	} else if len(failtxs) > 0 {
		var i int
		for i=0; i< len(failtxs); {
			walletbalance,_ := getWalletStatus()
			var ten_item_amount int64
			ten_item_amount = 0
			endindex := 0
			if i+10 > len(failtxs) {
				endindex = len(failtxs)
			} else {
				endindex = i + 10
			}
			for j:=i;j<endindex;j++ {
				ten_item_amount += failtxs[i].Amount
			}
			if walletbalance < ten_item_amount {
				Alert.Println("ReSendTxs:Wallet have no enough balance.")
				return 0
			}

			var txids []string
			for k:=i;k<endindex;k++ {
				txid, err := sendTx(failtxs[k].Amount, failtxs[k].Username)
				if err != nil {
					Warning.Println("ReSendTxs:sendTx to:", failtxs[k].Username, ",amount:", failtxs[k].Amount, "error:", err.Error())
					continue
				}
				data := bson.M{"$set": bson.M{"txstate": 100,"sendtime":time.Now().Unix()}}
				err = sendtx_t.UpdateId(failtxs[k].Id, data)
				if err != nil {
					Warning.Println("dealFailTx:senttx_t.Update txstate:", 100, "err:", err.Error())
				}
				sendonetx := &SendTx_t{Id: bson.NewObjectId(), TxId: txid, Username: failtxs[k].Username, Amount: failtxs[k].Amount, SendTime: time.Now().Unix(), TxState: 0, Worker: failtxs[k].Worker, ResendFlag: 1}
				err = sendtx_t.Insert(sendonetx)
				Info.Println("ReSendTxs:sendtx_t. before Insert usernanme:", failtxs[k].Username, ",workname:", failtxs[k].Worker, ",amount:", failtxs[k].Amount)
				if err != nil {
					Warning.Println("ReSendTxs:sendtx_t.Insert usernanme:", failtxs[k].Username, ",workname:", failtxs[k].Worker, ",amount:", failtxs[k].Amount, "error:", err.Error())
					continue
				}
				txids = append(txids, txid)
			}

			time.Sleep(time.Minute * 10)

			for _,item := range txids {
				txstate,err := getTxState(item)
				if err != nil {
					Warning.Println("ReSendTxs:getTxState. txid:", item, "error:", err.Error())
					continue
				}
				if txstate == 1 {
					success,err := cancelTx(item)
					if err != nil {
						Warning.Println("ReSendTxs:cancelTx txid:", item, "error:", err.Error())
						continue
					}
					if success {
						selector := bson.M{"txid": item}
						data := bson.M{"$set": bson.M{"txstate": 4}}
						err := sendtx_t.Update(selector, data)
						if err != nil {
							Warning.Println("ReSendTxs:sendtx_t.Update:", 4, "err:", err)
							return -1
						}
					}
				}
			}

			i += 10
		}
	}

	return 0
}

func (pool *Pool) pushMinerInfo(session *mgo.Session, statTime int64) {
	Info.Println("pushMinerInfo start.statTime:", statTime)
	pool.minersLock.RLock()
	defer pool.minersLock.RUnlock()

	share_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.ShareCol)
	share_t_totalnum, err := share_t.Count()
	if share_t_totalnum == 0 || err != nil {
		Warning.Println("pushMinerInfo:share_t collection is empty or get count error.:", err)
		return
	}

	var shares []Share_t
	share_t.Find(bson.M{"stime": bson.M{"$gte": statTime, "$lt": statTime + cfg.RewardPeriod}}).All(&shares)

	totalShare := 0
	var shares_map map[string]int64
	shares_map = make(map[string]int64)
	for _, item := range shares {
		if item.Valid == true {
			totalShare += 1
		}
		_, ok := shares_map[item.Username+"."+item.Worker]
		if ok {
			if item.Valid == true {
				shares_map[item.Username+"."+item.Worker] += 1
			} else {
				shares_map[item.Username+"."+item.Worker] += 0
			}
		} else {
			if item.Valid == true {
				shares_map[item.Username+"."+item.Worker] = 1
			} else {
				shares_map[item.Username+"."+item.Worker] = 0
			}
		}
	}

	foundblock_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.BlockCol)
	Info.Println("pushMinerInfo stat time:", statTime, "statendtime:", statTime+cfg.RewardPeriod)
	totalblocknum, err := foundblock_t.Find(bson.M{"stime": bson.M{"$gte": statTime, "$lt": statTime + cfg.RewardPeriod}}).Count()
	if err != nil {
		Warning.Println("foundblock_t collection is empty or get count error.:", err)
		return
	}

	var totalReward int64
	totalReward = cfg.OneBlockReward * int64(totalblocknum)
	minerinfo_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.MinerInfo)

	if shares == nil {
		minerinfo := &MinerInfo_t{Id: bson.NewObjectId(), Username: "", Reward: 0, StatTime: statTime + cfg.RewardPeriod, Worker: ""}
		err := minerinfo_t.Insert(minerinfo)
		if err != nil {
			Warning.Println("writemdb MinerInfo_t error:", err)
			return
		}
	} else {
		for k, v := range shares_map {
			names := strings.SplitN(k, ".", 2)
			if len(names) == 2 {
				username := names[0]
				username = strings.ToLower(username)
				workername := names[1]
				var reward int64
				reward = 0
				if totalShare > 0 {
					reward = v * totalReward / int64(totalShare)
				}
				minerinfo := &MinerInfo_t{Id: bson.NewObjectId(), Username: username, Reward: reward, StatTime: statTime + cfg.RewardPeriod, Worker: workername}
				err := minerinfo_t.Insert(minerinfo)
				if err != nil {
					Warning.Println("writemdb MinerInfo_t error:", err)
					return
				}
			}
		}
	}
}

func (pool *Pool) PushShareService() {
	pushTimer := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-pushTimer.C:
			session := pool.GetSession()
			if session == nil {
				Alert.Fatalln("GetSession is nil")
			} else {
				pool.pushShares(session)
				pool.pushFoundBlock(session)
				session.Close()
			}
		}
	}
}

func (pool *Pool) CalcUserReward() {
	pushTimer := time.NewTicker(time.Second * 300)
	for {
		select {
		case <-pushTimer.C:
			session := pool.GetSession()
			if session == nil {
				Alert.Fatalln("GetSession is nil")
			} else {
				timenow := time.Now().Unix()
				stattime := pool.isEmptyMinerInfo(session)
				if stattime <= 0 || stattime+cfg.RewardPeriod > timenow {
					session.Close()
					time.Sleep(20 * time.Second)
					continue
				}
				pool.pushMinerInfo(session, stattime)
				session.Close()
			}
		}
	}
}

func SendRewards(pool *Pool) {
	Info.Println("SendRewards.")
	session := pool.GetSession()
	if session == nil {
		Alert.Fatalln("GetSession is nil")
	} else {
		pool.UpdateAddrBalance(session)
		pool.SendReward(session)
		session.Close()
	}
}

func ReSendTx(pool *Pool) {
	Info.Println("ReSendTx.")
	session := pool.GetSession()
	if session == nil {
		Alert.Fatalln("GetSession is nil")
	} else {
		pool.ReSendTxs(session)
		session.Close()
	}
}

func (pool *Pool) SendRewardToUsers() {
	Info.Println("SendRewardToUsers.")

	gocron.Every(1).Day().At(cfg.SendRewardsTime).Do(SendRewards, pool)
	gocron.Every(1).Day().At(cfg.ReSendTime).Do(ReSendTx, pool)
	<-gocron.Start()
}

func (pool *Pool) CheckTxs(session *mgo.Session) {
	Info.Println("CheckTxs start.")

	start := time.Now()

	sendtx_t := session.DB(pool.cfg.MongoDB.DBname).C(pool.cfg.MongoDB.SendTx)
	var txinfos []SendTx_t
	sendtx_t.Find(bson.M{"txstate": bson.M{"$in": []int64{0, 1, 5, -1}}}).All(&txinfos)
	totalnum, err := sendtx_t.Find(bson.M{"txstate": bson.M{"$in": []int64{0, 1, 5, -1}}}).Count()
	if err != nil {
		Warning.Println("CheckTxs:sendtx_t collection get count error:", err)
		return
	}

	if totalnum == 0 {
		Warning.Println("CheckTxs:sendtx_t collection is have no records state in(0,1,5),do not need update.")
		return
	} else if totalnum > 0 {
		for _, item := range txinfos {
			state, err := getTxState(item.TxId)
			if err != nil {
				Warning.Println("CheckTxs:getTxState err:", err)
			}
			if state != item.TxState {
				data := bson.M{"$set": bson.M{"txstate": state}}
				err := sendtx_t.UpdateId(item.Id, data)
				if err != nil {
					Warning.Println("CheckTxs:sendtx_t.Update:", state, "err:", err)
				}
			}
		}
	}

	Info.Println("CheckTxs finished:", time.Since(start))
}

func (pool *Pool) CheckTxState() {
	pushTimer := time.NewTicker(time.Second * 360)
	for {
		select {
		case <-pushTimer.C:
			session := pool.GetSession()
			if session == nil {
				Alert.Fatalln("GetSession is nil")
			} else {
				pool.CheckTxs(session)
				session.Close()
			}
		}
	}
}
