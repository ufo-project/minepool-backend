package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

type getUtxoRequest struct {
	JsonRpc string `json:"jsonrpc"`
	Id      int64  `json:"id"`
	Method  string `json:"method"`
}

type getUtxoResponse struct {
	JsonRpc string          `json:"jsonrpc"`
	Id      int64           `json:"id"`
	Result  []getUtxoResult `json:"result,omitempty"`
	Error   ErrorInfo       `json:"error,omitempty"`
}

type getUtxoResult struct {
	Amount        int64  `json:"amount"`
	CreateTxId    string `json:"createTxId"`
	Id            string `json:"id"`
	Maturity      int64  `json:"maturity"`
	Session       int64  `json:"session"`
	SpentTxId     string `json:"spentTxId"`
	Status        int64  `json:"status"`
	Status_string string `json:"status_string"`
	Type          string `json:"type"`
}

func GetUtxos() (balance int64, err error) {
	var reqUtxo getUtxoRequest
	reqUtxo.Method = "get_utxo"
	reqUtxo.JsonRpc = "2.0"
	reqUtxo.Id = 1

	client := &http.Client{}
	bytesData, _ := json.Marshal(reqUtxo)
	req, _ := http.NewRequest("POST", cfg.WalletApiUrl, bytes.NewReader(bytesData))
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

	var res getUtxoResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		Warning.Println("Unmarshal Response error:", err)
		return 0, err
	}

	if res.Error.Message != "" {
		err = errors.New(res.Error.Message)
		return 0, err
	}

	var totalAmount int64
	for _, item := range res.Result {
		if item.Type != "mine" {
			continue
		}

		if item.Status != 2 && item.Status != 1 {
			continue
		}
		amount := item.Amount

		totalAmount += amount
	}

	return totalAmount, nil
}

type sendTxRequest struct {
	JsonRpc string       `json:"jsonrpc"`
	Id      int64        `json:"id"`
	Method  string       `json:"method"`
	Params  sendtxParams `json:"params"`
}

type sendtxParams struct {
	Value   int64  `json:"value"`
	Fee     int64  `json:"fee,omitempty"`
	From    string `json:"from,omitempty"`
	Address string `json:"address"`
	Comment string `json:"comment,omitempty"`
}

type sendTxResponse struct {
	JsonRpc string       `json:"jsonrpc"`
	Id      int64        `json:"id"`
	Result  sendtxResult `json:"result,omitempty"`
	Error   ErrorInfo    `json:"error,omitempty"`
}

type sendtxResult struct {
	Txid string `json:"txId"`
}

type ErrorInfo struct {
	Code    int64  `json:"code"`
	Data    string `json:"data"`
	Message string `json:"message"`
}

func sendTx(amount int64, toaddr string) (txId string, err error) {
	client := &http.Client{}
	var reqSendtx sendTxRequest
	reqSendtx.JsonRpc = "2.0"
	reqSendtx.Id = 1
	reqSendtx.Method = "tx_send"
	var reqparams sendtxParams
	reqparams.Value = amount
	reqparams.Address = toaddr
	reqSendtx.Params = reqparams
	bytesData, err := json.Marshal(reqSendtx)
	if err != nil {
		Warning.Println("Marshal reqSendtx error:", err.Error())
	}
	Info.Println("bytesData:", string(bytesData))
	req, err := http.NewRequest("POST", cfg.WalletApiUrl, bytes.NewReader(bytesData))
	if err != nil {
		Warning.Println("NewRequest error:", err.Error())
	}
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

	var res sendTxResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		Warning.Println("Unmarshal Response error:", err.Error())
		return "", err
	}

	if res.Error.Message != "" {
		err = errors.New(res.Error.Message)
		return "", err
	}

	if res.Result.Txid != "" {
		return res.Result.Txid, nil
	}
	return
}

type validAddrRequest struct {
	JsonRpc string          `json:"jsonrpc"`
	Id      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  validAddrParams `json:"params"`
}

type validAddrParams struct {
	Address string `json:"address"`
}

type validAddrResponse struct {
	JsonRpc string          `json:"jsonrpc"`
	Id      int64           `json:"id"`
	Result  validAddrResult `json:"result,omitempty"`
	Error   ErrorInfo       `json:"error,omitempty"`
}

type validAddrResult struct {
	IsMine  bool `json:"is_mine"`
	IsValid bool `json:"is_valid"`
}

func validAddrress(addr string) (isvalid bool, err error) {
	client := &http.Client{}
	var validaddr validAddrRequest
	validaddr.JsonRpc = "2.0"
	validaddr.Id = 1
	validaddr.Method = "validate_address"
	var reqparams validAddrParams
	reqparams.Address = addr
	validaddr.Params = reqparams
	bytesData, _ := json.Marshal(validaddr)
	req, _ := http.NewRequest("POST", cfg.WalletApiUrl, bytes.NewReader(bytesData))
	resp, _ := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	} else {
		err = errors.New("Connect wallet-api faild,please check.")
		return false, err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

	var res validAddrResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		Warning.Println("Unmarshal Response error:", err)
		return false, err
	}

	if res.Error.Message != "" {
		err = errors.New(res.Error.Message)
		return false, err
	}

	return res.Result.IsValid, nil
}

type checkTxRequest struct {
	JsonRpc string        `json:"jsonrpc"`
	Id      int64         `json:"id"`
	Method  string        `json:"method"`
	Params  checkTxParams `json:"params"`
}

type checkTxParams struct {
	TxId string `json:"txId"`
}

type checkTxResponse struct {
	JsonRpc string        `json:"jsonrpc"`
	Id      int64         `json:"id"`
	Result  checkTxResult `json:"result,omitempty"`
	Error   ErrorInfo     `json:"error,omitempty"`
}

type checkTxResult struct {
	TxId           string `json:"txId"`
	Comment        string `json:"comment"`
	Fee            int64  `json:"fee"`
	Kernel         string `json:"kernel"`
	Receiver       string `json:"receiver"`
	Sender         string `json:"sender"`
	Status         int64  `json:"status"`
	Status_string  string `json:"status_string"`
	Failure_reason string `json:"failure_reason"`
	Value          int64  `json:"value"`
	Height         int64  `json:"height"`
	Confirmations  int64  `json:"confirmations"`
	Create_time    int64  `json:"create_time"`
	Income         bool   `json:"income"`
}

func getTxState(txid string) (state int64, err error) {
	client := &http.Client{}
	var checkTx checkTxRequest
	checkTx.JsonRpc = "2.0"
	checkTx.Id = 1
	checkTx.Method = "tx_status"
	var reqparams checkTxParams
	reqparams.TxId = txid
	checkTx.Params = reqparams
	bytesData, _ := json.Marshal(checkTx)
	req, _ := http.NewRequest("POST", cfg.WalletApiUrl, bytes.NewReader(bytesData))
	resp, _ := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	} else {
		err = errors.New("Connect wallet-api faild,please check.")
		return -1, err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

	var res checkTxResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		Warning.Println("Unmarshal Response error:", err)
		return -1, err
	}

	if res.Error.Message != "" {
		err = errors.New(res.Error.Message)
		return -1, err
	}

	return res.Result.Status, nil
}
