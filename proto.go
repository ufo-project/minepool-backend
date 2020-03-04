package main

import (
	"sync"
)

//For miner
type MinerRequest struct {
	Id        string        `json:"id"`
	Method    string        `json:"method"`
	Minertype string        `json:"minertype"`
	Miner     string        `json:"miner"`
	JobID     string        `json:"jobid"`
	Nonce     string        `json:"nonce"`
	JsonRPC   string        `json:"jsonrpc,omitempty"`
	Params    []interface{} `json:"params"`
}

type MinerRequest2 struct {
	Id        int           `json:"id"`
	Method    string        `json:"method"`
	Minertype string        `json:"minertype"`
	Miner     string        `json:"miner"`
	JobID     string        `json:"jobid"`
	Nonce     string        `json:"nonce"`
	JsonRPC   string        `json:"jsonrpc,omitempty"`
	Params    []interface{} `json:"params"`
}

//From wallet
type Response struct {
	Id      string `json:"id"`
	Method  string `json:"method"`
	JsonRPC string `json:"jsonrpc,omitempty"`

	//For job
	NBits    uint32 `json:"nbits,omitempty"`
	Height   int64  `json:"height,omitempty"`
	Input    string `json:"input,omitempty"`
	PrevHash string `json:"prev,omitempty"`

	//For login
	Code        int    `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
	ForkHeight  int    `json:"forkheight,omitempty"`
}

type Response2 struct {
	Id      int    `json:"id"`
	Method  string `json:"method"`
	JsonRPC string `json:"jsonrpc,omitempty"`

	//For job
	NBits    uint32 `json:"nbits,omitempty"`
	Height   int64  `json:"height,omitempty"`
	Input    string `json:"input,omitempty"`
	PrevHash string `json:"prev,omitempty"`

	//For login
	Code        int    `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
	ForkHeight  int    `json:"forkheight,omitempty"`
}

type ShareSet struct {
	shareLock sync.Mutex
	shares    map[string]struct{}
}
