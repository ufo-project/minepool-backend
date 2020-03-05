package main

import (
	"time"

	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/jrick/bitset"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/mgo.v2"
	"math/big"
	"net"
	"os"
	"sync"
)

const (
	HashSize = 32
)

type Pool struct {
	Coin    string
	IP      string
	version string
	isLogin bool

	cfg Config

	levelDB      *leveldb.DB
	mgoSession   *mgo.Session
	share_t      *mgo.Collection
	foundblock_t *mgo.Collection

	found_block_mapLock sync.Mutex
	found_block_map     map[string]string

	WalletUrl    string
	WalletApiUrl string
	conn         net.Conn
	connbuff     *bufio.Reader

	height      int64
	isFirstWork bool
	ready       bool
	walletReady chan bool
	walletLock  sync.RWMutex

	JobsLock   sync.RWMutex
	Jobs       map[string]Job
	LastJob    Job
	LastNotify string
	ForkHeight int

	StartTarget *big.Int
	StartDiff4G float64

	lastNewBlockTime time.Time
	interval         time.Duration

	minersLock  sync.RWMutex
	miners      map[*Miner]struct{}
	preEnonce   int
	enonceRange int
	enonce      bitset.Bytes

	shareSet ShareSet

	timeout time.Duration

	staleCount     int64
	duplicateCount int64
	lowDiffCount   int64
	acceptCount    int64

	blockCount int

	shareDaily float64

	WalletCheckInterval time.Duration

	SubmitSol chan Response
	ValidAddress map[string]bool
}

func (pool *Pool) ListenTCP() {
	addr, err := net.ResolveTCPAddr("tcp", pool.cfg.PoolIPPort)
	if err != nil {
		Alert.Fatalf("ResolveTCPAddr error: %v", err)
	}
	server, err := net.ListenTCP("tcp", addr)
	if err != nil {
		Alert.Fatalf("ListenTCP error: %v", err)
	}
	defer server.Close()

	Info.Printf("listening on %s", pool.cfg.PoolIPPort)

	var accept = make(chan int, pool.enonceRange)
	n := 0

	for {
		conn, err := server.AcceptTCP()
		if err != nil {
			continue
		}
		conn.SetKeepAlive(true)

		ip, port, _ := net.SplitHostPort(conn.RemoteAddr().String())

		n += 1
		miner := &Miner{conn: conn, IP: ip, Port: port}

		accept <- n
		go func(miner *Miner) {
			miner.pool = pool
			err := miner.handleTCPClient()

			pool.removeMiner(miner)
			conn.Close()

			if miner.isAuthorize {
				Info.Printf("RemoveMiner %s.%s error:%v\n", miner.username, miner.workername, err)
			}
			<-accept
		}(miner)
	}
}

func (pool *Pool) addMiner(miner *Miner) {
	pool.minersLock.Lock()
	defer pool.minersLock.Unlock()

	miner.ENonce, miner.EnonceNum = pool.generateENonce()
	pool.miners[miner] = struct{}{}

}

func (pool *Pool) removeMiner(miner *Miner) {
	pool.minersLock.Lock()
	defer pool.minersLock.Unlock()

	if _, found := pool.miners[miner]; found {
		pool.reuseENonce(miner.EnonceNum)
		delete(pool.miners, miner)
	}
}

func (pool *Pool) generateENonce() (string, int) {
	var enonce int
	for enonce = 0; enonce < pool.enonceRange; enonce++ {
		if !pool.enonce.Get(enonce) {
			break
		}
	}

	if enonce < pool.enonceRange {
		pool.enonce.Set(enonce)

		enonce = pool.preEnonce + enonce

		if cfg.ENonceLen == 2 {
			return fmt.Sprintf("%04x", enonce), enonce

		} else if cfg.ENonceLen == 3 {
			return fmt.Sprintf("%06x", enonce), enonce

		} else if cfg.ENonceLen == 4 {
			return fmt.Sprintf("%08x", enonce), enonce
		}

	} else {
		Warning.Println("generateENonce error: enonce exhaust")
	}

	return "", 0
}

func (pool *Pool) reuseENonce(enonce int) {
	pool.enonce.Unset(enonce)
}

func (pool *Pool) PrintDailyShareServer() {

	daySeconds := int64(86400)
	now := time.Now()
	time8hour := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())

	diff := time8hour.Unix() - now.Unix()

	if diff < 0 {
		diff = daySeconds + diff
	}

	Info.Println("PrintDailyShareServer will start timer after", diff, "second")

	dailyTimer := time.NewTimer(time.Second * time.Duration(diff))
	for {
		select {
		case <-dailyTimer.C:

			dailyTimer.Reset(time.Second * time.Duration(daySeconds))
			pool.PrintDailyShare(float64(diff))
			pool.shareDaily = 0
			diff = 86400
		}
	}
}

func (pool *Pool) PrintDailyShare(sec float64) {

	hashRate := pool.shareDaily * UnitChange / sec
	BlockLog.Println("Daily share", hashRate, "GH/s")
}

func CalcPowHash(data string, poolTarget, netTarget *big.Int) (bool, bool, string) {
	isValidShare := false
	isValidBlock := false

	var powHash Hash
	hash := X17r_Sum256(data)
	copy(powHash[:], hash)
	hashBig := HashToBig(&powHash)

	Info.Println("header:", data)
	Info.Println("hash:", hex.EncodeToString(hash))
	Info.Println("poolTarget:", fmt.Sprintf("%064s", hex.EncodeToString(poolTarget.Bytes())))
	Info.Println("netTarget:", fmt.Sprintf("%064s", hex.EncodeToString(netTarget.Bytes())))
	Info.Println("hashBig:", fmt.Sprintf("%064s", hex.EncodeToString(hashBig.Bytes())))

	if hashBig.Cmp(poolTarget) <= 0 {
		isValidShare = true
		if hashBig.Cmp(netTarget) <= 0 {
			isValidBlock = true
		}
	}

	return isValidShare, isValidBlock, fmt.Sprintf("%064s", hex.EncodeToString(hashBig.Bytes()))
}

func TestPow() {
	Info.Println("TestPow+")
	var powHash Hash
	data := "00000000b4f0a89aa1ca0da9fddd326bcb41cc7864e602252b07287b204cd4ccef01000000000000f5c8b2a75e9ac4de3a48bb80bd6d1a2f78c87572b88d9c2109e745e349bc4dfe00004c00a989d3c0"
	hash := X17r_Sum256(data)
	copy(powHash[:], hash)
	hashBig := HashToBig(&powHash)

	Info.Println("header:", data)
	Info.Println("hash:", hex.EncodeToString(hash))
	Info.Println("diff:", new(big.Int).Div(pow256, hashBig).String())
	Info.Println("hash:", hash)
	os.Exit(1)
}
