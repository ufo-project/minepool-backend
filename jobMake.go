package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"time"
)

func (pool *Pool) updateWork() {
	pool.isLogin = false
	pool.ConnectWallet()
	pool.Login()

	for {
		line, isPrefix, err := pool.connbuff.ReadLine()
		if isPrefix {
			Warning.Printf("Socket flood detected from %s", pool.WalletUrl)

		} else if err != nil {
			Warning.Println("pool", pool.IP, "error:", err)
			pool.conn.Close()
			pool.conn = nil
			pool.ConnectWallet()
			pool.Login()
			continue
		}

		if len(line) > 1 {
			Info.Println(string(line))

			var resp Response
			err := json.Unmarshal(line, &resp)
			if err != nil {
				Warning.Println("Unmarshal Response error:", err)
				continue
			}
			if resp.Method == "job" {
				pool.handleGetWork(resp.Id, resp.PrevHash, resp.Input, resp.Height, resp.NBits)

			} else if resp.Method == "result" {
				if resp.Id == "login" {
					Info.Println("login")
					gPool.ForkHeight = resp.ForkHeight

				} else {
					if resp.Description == "accepted" || resp.Description == "expired" || resp.Description == "rejected" {
						pool.SubmitSol <- resp
						if resp.Description == "expired" {
							pool.conn.Close()
							pool.conn = nil
							pool.ConnectWallet()
							pool.Login()
							continue
						}
					}
					Info.Println("result")
				}
			}
		}
	}
}

var (
	big100 = new(big.Int).SetInt64(100)
	big99  = new(big.Int).SetInt64(99)
)

func (pool *Pool) handleGetWork(id, prevHash, input string, height int64, nbits uint32) {
	if height < pool.height {
		Warning.Printf("Slow block at height %d, current height:%d, input:%s", height, pool.height, input)
		return
	}

	if height > pool.height {
		if height > cfg.HalveHeight {
			cfg.OneBlockReward /= 2
		}
		go pool.broadcastNewJobs(id, prevHash, input)

		pool.shareSet.shareLock.Lock()
		pool.shareSet.shares = make(map[string]struct{})
		pool.shareSet.shareLock.Unlock()

		target := CompactToBig(nbits)
		targetReal := new(big.Int).Div(new(big.Int).Mul(target, big100), big99)
		netDiff := new(big.Int).Div(pow256, target).Int64()
		diffstr := ToFloat(nbits)
		currDiffStr := fmt.Sprintf("%.6f", float64(diffstr))
		netDiffStr := fmt.Sprintf("%f", float64(netDiff)/4/1024/1024/1024)

		pool.JobsLock.Lock()
		defer pool.JobsLock.Unlock()

		pool.Jobs = make(map[string]Job)
		pool.LastJob = Job{
			Input:       input,
			Height:      height,
			Id:          id,
			PrevHash:    prevHash,
			Target:      targetReal,
			netDiffStr:  netDiffStr,
			currDiffStr: currDiffStr,
		}
		pool.Jobs[id] = pool.LastJob
		pool.height = height
		pool.isFirstWork = true

		Info.Printf("New block %d :%s", height, input)
		Info.Printf("nbits:0x%x, target:%d, targetReal:%d, netDiff:%d, %s", nbits, target, targetReal, netDiff, netDiffStr)

	}

	if !pool.ready {
		pool.ready = true
		pool.walletReady <- true
	}
}

func (pool *Pool) ConnectWallet() {
	var retry int

	for {
		conn, err := net.Dial("tcp", pool.WalletUrl)
		if err == nil {
			pool.conn = conn
			pool.connbuff = bufio.NewReaderSize(pool.conn, 1024)
			Info.Println("Connect Agent", pool.WalletUrl, "success")
			return
		}

		retry++
		if retry%10 == 0 {
			Info.Println("Dial", pool.WalletUrl, retry+1, "times,error:", err)
		}
		if retry > 100 {
			Alert.Panicln("Dial", pool.WalletUrl, "failed over 100 times, exit...")
		}
		time.Sleep(time.Second)
	}
}

func (pool *Pool) SubmitBlock(data string) {

	for {
		if pool.conn != nil {
			break
		}
		time.Sleep(time.Second)
	}

	_, err := pool.conn.Write([]byte(data))
	if err != nil {
		BlockLog.Println("SubmitBlock error:", err)
	}
}

type Job struct {
	Input    string
	Height   int64
	Id       string
	PrevHash string

	Target      *big.Int
	netDiffStr  string
	currDiffStr string
}

func (pool *Pool) broadcastNewJobs(jobid, prehash, input string) {
	pool.minersLock.RLock()
	defer pool.minersLock.RUnlock()

	start := time.Now()
	bcast := make(chan int, 512)
	n := 0

	for m, _ := range pool.miners {
		n++
		bcast <- n

		go func(miner *Miner) {
			newDiff := miner.dc.calcCurDiff()

			miner.Lock()
			if miner.diff != newDiff {
				Info.Println("diff:", miner.diff, "newdiff:", newDiff)

				miner.diff = newDiff
				miner.diffStr = fmt.Sprintf("%f", miner.diff)
				miner.target = new(big.Int).Div(pow256, new(big.Int).SetInt64(int64(miner.diff*DiffUnit)))

				miner.SetDifficulty()
			}

			var respStr string
			if miner.minertype == "cpu" {
				respStr = fmt.Sprintf("{\"jobid\":\"%s\",\"prev\":\"%s\",\"input\":\"%s\",\"id\":\"1\",\"jsonrpc\":\"2.0\",\"method\":\"mining_notify\"}\n", jobid, prehash, input)
			} else {
				respStr = fmt.Sprintf("{\"id\":null,\"method\":\"mining.notify\",\"params\":[\"%s\",\"%s\",\"%s\",true]}\n", jobid, prehash, input)
			}
			pool.LastNotify = respStr
			_, err := miner.conn.Write([]byte(respStr))
			miner.Unlock()

			if err != nil {
				Warning.Printf("BroadcastNewJobs to %s.%s@%s error: %v", miner.username, miner.workername, miner.IP, err)
			}

			<-bcast
		}(m)
	}
	pool.lastNewBlockTime = time.Now()

	Info.Printf("Broadcast Jobs to %d miner finished %s", n, time.Since(start))
}

func (pool *Pool) Login() {
	loginCmd := fmt.Sprintf("{\"method\":\"login\",\"api_key\":\"%s\",\"id\":\"login\",\"jsonrpc\":\"2.0\"}\n", cfg.ApiKey)
	_, err := pool.conn.Write([]byte(loginCmd))
	if err != nil {
		Info.Println("Login error:", err)
	}
}
