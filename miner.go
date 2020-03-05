package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"fmt"
	"math/big"
	"strconv"
)

var (
	CurrentBLockNull = errors.New("currentBlockTemplate nil")
	SendJsonToMiner  = errors.New("send json to miner err")
)

var (
	hashPattern   = regexp.MustCompile("^0x[0-9a-f]{64}$")
	workerPattern = regexp.MustCompile("^[0-9a-zA-Z-_]{1,8}$")
)

const (
	STALE_SHARE = 3

	JOB_NOT_FOUND   = 21
	DUPLICATE_SHARE = 22

	LOW_DIFFICULTY = 23
	NOT_LOGIN      = 24
	NOT_GETWORK    = 25

	ILLEGAL_PARARMS = 27

	UNKNOWN = 50
)

type Miner struct {
	sync.Mutex

	ENonce    string
	EnonceNum int

	username   string
	workername string
	minertype  string

	IP   string
	Port string

	conn *net.TCPConn
	enc  *json.Encoder

	diff    float64
	diffStr string
	target  *big.Int

	dc *diffController

	isSubscribe bool
	isAuthorize bool

	pool *Pool

	onlineTime time.Time

	acceptNum    int
	rejectNum    int
	staleNum     int
	duplicateNum int

	acceptShare float64
}

type Name struct {
	username   string
	workername string
}

func (miner *Miner) setDeadline() {
	miner.conn.SetDeadline(time.Now().Add(miner.pool.timeout))
}

func (miner *Miner) handleTCPClient() error {
	miner.enc = json.NewEncoder(miner.conn)

	connbuff := bufio.NewReaderSize(miner.conn, 1024)
	miner.setDeadline()

	for {
		line, isPrefix, err := connbuff.ReadLine()
		if isPrefix {
			Warning.Printf("Socket flood detected from %s", miner.IP)
			return err

		} else if err == io.EOF {
			return err

		} else if err != nil {
			return err
		}
		miner.setDeadline()

		if len(line) > 1 {
			Info.Println(string(line))

			var req MinerRequest
			var req2 MinerRequest2
			err = json.Unmarshal(line, &req)
			if err != nil {
				err = json.Unmarshal(line, &req2)
				if err != nil {
					if miner.isAuthorize {
						Warning.Println("Unmarshal error:", err, "line:", string(line))
					}
					return err
				}
				req.Id = strconv.Itoa(req2.Id)
				req.Method = req2.Method
				req.Params = req2.Params
				req.Nonce = req2.Nonce
				req.JobID = req2.JobID
				req.Miner = req2.Miner
				req.Minertype = req2.Minertype
				req.JsonRPC = req2.JsonRPC
			}

			start := time.Now()
			err = miner.handleTCPMessage(&req)
			if err != nil {
				if miner.isAuthorize {
					Warning.Println(miner.username, "handleTCPMessage error:", err, "elaspe:", time.Since(start))
				}
				return err
			}
		}
	}
	return nil
}

func (miner *Miner) handleTCPMessage(req *MinerRequest) error {
	Info.Println("handleTCPMessage.")
	var err error
	switch req.Method {
	case "mining_subscribe":
		err = miner.subscribeHandle(req)
	case "mining_authorize":
		err = miner.authorizeHandle(req)
	case "mining_submit":
		err = miner.submitHandle(req)
	default:
		Info.Println(miner.workername, "  ", miner.IP, " miner reqest "+req.Method+" not implemented!")
	}

	return err
}

func (miner *Miner) checkShare(shareStr string) error {

	miner.pool.shareSet.shareLock.Lock()
	defer miner.pool.shareSet.shareLock.Unlock()

	_, exist := miner.pool.shareSet.shares[shareStr]
	if exist {
		miner.duplicateNum++
		miner.pool.duplicateCount++
		Info.Printf("%s.%s duplicate share", miner.username, miner.workername)

		return errors.New("duplicate share")

	} else {
		miner.pool.shareSet.shares[shareStr] = struct{}{}
	}
	return nil
}

func (miner *Miner) submitHandle(req *MinerRequest) error {
	var header, blockhash string
	var valid = "0"
	var isValidShare, isValidBlock, found bool
	var job Job

	if !miner.isAuthorize {
		Info.Println("don't Authorize")
		return errors.New("don't Authorize")
	}

	var jobID, nonce string
	jobID = req.JobID
	nonce = req.Nonce


	nonceStr := miner.ENonce + strings.ToLower(nonce)
	if len(nonceStr) != 16 {
		Warning.Println("len of nonce is not equal 16")
		return errors.New("len of nonce is not equal 16")
	}
	err := miner.checkShare(nonceStr)
	if err != nil {
		goto out
	}

	gPool.JobsLock.RLock()
	job, found = miner.pool.Jobs[jobID]
	gPool.JobsLock.RUnlock()

	if !found {
		Warning.Println("Stale share:", jobID)

		miner.pool2MinerError(req.Id, STALE_SHARE)
		miner.staleNum++
		miner.pool.staleCount++

		goto out
	}

	header = "00000000" + job.PrevHash + "00000000" + job.Input + miner.ENonce + nonce

	isValidShare, isValidBlock, blockhash = CalcPowHash(header, miner.target, job.Target)

	if isValidBlock {

		var submitResult bool

		blockStr := fmt.Sprintf("{\"id\":\"%s\",\"jsonrpc\":\"2.0\",\"method\":\"solution\",\"nonce\":\"%s\"}\n", jobID, nonceStr)
		miner.pool.SubmitBlock(blockStr)

		result := <-gPool.SubmitSol
		Info.Println("submit solution result:", result, "jobid", jobID, "result.id", result.Id, "description:", result.Description)
		if result.Id == jobID && result.Description == "accepted" {
			submitResult = true
		}

		if submitResult == true {
			BlockLog.Printf("blockhash:%s,height %d miner %s.%s@%s nonce:%s", blockhash, job.Height, miner.username, miner.workername, miner.IP, nonceStr)

			height := int64(job.Height)
			miner.pool.found_block_mapLock.Lock()
			miner.pool.found_block_map[job.Id] = miner.username + `|` + miner.workername + `|` + strconv.FormatInt(height, 10) + `|` + strconv.FormatInt(time.Now().Unix(), 10) + `|` + blockhash
			Info.Println("found_block_map blocks:", miner.pool.found_block_map[job.Id])
			miner.pool.found_block_mapLock.Unlock()

			miner.pool.blockCount++

			Info.Println("blocks:", miner.pool.blockCount)
		}
	}

	if isValidShare {
		miner.pool2SubmitMinerTrue(req.Id)

		valid = "1"

		miner.acceptNum++
		miner.pool.acceptCount++

		miner.acceptShare += miner.diff
		miner.pool.shareDaily += miner.diff
	} else {
		miner.pool2MinerError(req.Id, LOW_DIFFICULTY)

		miner.rejectNum++
		miner.pool.lowDiffCount++

		Info.Println(miner.username, ".", miner.workername, "Low difficulty")
	}

out:
	miner.dc.addShare()
	key := job.Input + miner.ENonce + nonce
	miner.insertShareToLevelDB(key, valid, job.netDiffStr)

	return nil
}

func (miner *Miner) subscribeHandle(req *MinerRequest) error {

	Info.Println("subscribeHandle.")
	miner.diff = miner.pool.StartDiff4G
	miner.target = miner.pool.StartTarget

	miner.diffStr = fmt.Sprintf("%f", miner.diff)

	miner.isSubscribe = true
	miner.dc = newDiffController(miner.diff)
	miner.minertype = req.Minertype

	miner.pool.addMiner(miner)

	miner.onlineTime = time.Now()

	err := miner.SubscribeResponse(req.Id, miner.ENonce)

	if err != nil {
		return err
	}
	Info.Printf("miner:%s:%s enonce:%s subscribe", miner.IP, miner.Port, miner.ENonce)

	return nil
}

func (miner *Miner) authorizeHandle(req *MinerRequest) error {

	if len(req.Params) < 1 && req.Miner == "" {
		Info.Println("submitHandle param len < 1")
		return errors.New("submitHandle param len < 1")
	}

	name := ""
	name = req.Miner

	usernameLower := strings.ToLower(name)
	username := strings.Replace(usernameLower, "0x", "", -1)

	SplitName := strings.Split(username, ".")
	if len(SplitName) > 1 {
		miner.username = SplitName[0]
		miner.workername = SplitName[1]
	}

	ifvalid, exist := gPool.ValidAddress[miner.username]
	if !exist {
		isvalid, err := validAddrress(miner.username)

		if err != nil {
			Info.Println("authorizeHandle validAddrress,isvalid:", isvalid, ",error:", err)
			miner.pool2MinerTrue(1, req.Id)
			return err
		}
		if !isvalid {
			Info.Println("authorizeHandle validAddrress,isvalid:", isvalid, ",error:", err)
			miner.pool2MinerTrue(1, req.Id)
			gPool.ValidAddress[miner.username] = isvalid
			return err
		} else {
			gPool.ValidAddress[miner.username] = isvalid
		}
	} else {
		if ifvalid == false {
			Info.Println("authorizeHandle validAddrress has existed,isvalid:false")
			miner.pool2MinerTrue(1, req.Id)
			errors.New("")
			return errors.New("authorizeHandle validAddrress has existed,isvalid:false")
		}
	}

	miner.isAuthorize = true
	miner.minertype = req.Minertype

	miner.pool2MinerTrue(0, req.Id)
	miner.SetDifficulty()

	miner.pool.JobsLock.RLock()
	defer miner.pool.JobsLock.RUnlock()

	notifyStr := ""
	notifyStr = fmt.Sprintf("{\"jobid\":\"%s\",\"prev\":\"%s\",\"input\":\"%s\",\"id\":\"1\",\"jsonrpc\":\"2.0\",\"method\":\"mining_notify\"}\n", miner.pool.LastJob.Id, miner.pool.LastJob.PrevHash, miner.pool.LastJob.Input)

	miner.SendNotify(notifyStr)

	Info.Printf("welcome miner:%s.%s@%s:%s", miner.username, miner.workername, miner.IP, miner.Port)

	return nil
}

func (miner *Miner) pool2MinerTrue(code int, id string) error {
	msg := ""
	msg = fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":\"%s\",\"code\":%d,\"method\":\"mining_authorize_result\"}\n", id, code)

	return miner.SendMessageToMiner(msg)
}

func (miner *Miner) pool2SubmitMinerTrue(id string) error {
	msg := ""

	msg = fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":\"%s\",\"code\":0,\"method\":\"mining_submit_result\"}\n", id)

	return miner.SendMessageToMiner(msg)
}

func (miner *Miner) pool2MinerError(id string, code int) error {

	msg := ""

	msg = fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":\"%s\",\"code\":%d,\"method\":\"mining_submit_result\"}\n", id, code)

	return miner.SendMessageToMiner(msg)
}

func (miner *Miner) SubscribeResponse(id string, ENonce string) error {

	Info.Println("SubscribeResponse.")
	msg := ""

	code := 0
	msg = fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":\"%s\",\"code\":%d,\"enonce\":\"%s\",\"method\":\"mining_subscribe_result\"}\n", id, code, ENonce)

	return miner.SendMessageToMiner(msg)
}

func (miner *Miner) SendNotify(msg string) error {
	return miner.SendMessageToMiner(msg)
}

func (miner *Miner) SetDifficulty() error {

	msg := ""

	diffStr := fmt.Sprintf("%f", miner.diff*256)
	msg = fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":\"1\",\"difficulty\":\"%s\",\"method\":\"mining_set_difficulty\"}\n", diffStr)

	return miner.SendMessageToMiner(msg)
}

func (miner *Miner) SendMessageToMiner(msg string) error {
	_, err := miner.conn.Write([]byte(msg))
	Info.Println("SendMessageToMiner:", msg)
	if err != nil {
		Info.Println("SendMessageToMiner failed")
	}
	return err
}
