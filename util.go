package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

func BigPow(a, b int64) *big.Int {
	r := big.NewInt(a)
	return r.Exp(r, big.NewInt(b), nil)
}

func GetTargetHex(diff *big.Int) string {

	target := new(big.Int).Div(pow256, diff)

	hex := hex.EncodeToString(target.Bytes())

	return "0x" + fmt.Sprintf("%064s", hex)

}

func GetBytesFromHex(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}

	h, _ := hex.DecodeString(s)
	return h
}

func TargetHexToBigDiff(targetHex string) *big.Int {
	targetBytes := GetBytesFromHex(targetHex)
	return new(big.Int).Div(pow256, new(big.Int).SetBytes(targetBytes))
}

func GetPublicIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		Warning.Println("getPublicIP error:", err)
		return "default"
	}

	var localIP net.IP
	var publicIP net.IP

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				if ip4[0] == 10 || ip4[0] == 192 {
					localIP = ip4
				} else {
					publicIP = ip4
				}
			}
		}
	}
	if publicIP == nil {
		publicIP = localIP
	}
	if publicIP == nil {
		return "default"
	}

	return publicIP.String()

}

func ParseDuration(s string) time.Duration {
	value, err := time.ParseDuration(s)
	if err != nil {
		panic("util: Can't parse duration `" + s + "`: " + err.Error())
	}
	return value
}

func MkdirIfNoExist(dir string) error {
	_, err := os.Stat(dir)

	if os.IsNotExist(err) {
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			log.Println("mkdir failed![%v]\n", err)
		} else {
			log.Println("mkdir success!\n")
		}
		return err
	}

	return err
}

var (
	kilo float64 = 1000
	mega float64 = 1000000
	giga float64 = 1000000000
	tera float64 = 1000000000000
)

func PrintHashRateSuffix(hashrate int) {
	var shareStr string
	rate := float64(hashrate)
	if rate > tera {
		val := rate / tera
		shareStr = fmt.Sprintf("pool hashrate:%.3fEH/s", val)
	} else if rate > giga {
		val := rate / giga
		shareStr = fmt.Sprintf("pool hashrate:%.3fGH/s", val)
	} else if rate > mega {
		val := rate / mega
		shareStr = fmt.Sprintf("pool hashrate:%.3fMH/s", val)
	} else if rate > kilo {
		val := rate / kilo
		shareStr = fmt.Sprintf("pool hashrate:%.3fKH/s", val)
	} else {
		shareStr = fmt.Sprintf("pool hashrate:%dH/s", hashrate)
	}
	ShareLog.Println(shareStr)
}

func GetErrorCodeString(errorCode int) string {
	var ret string
	switch errorCode {
	case STALE_SHARE:
		ret = "Stale share"
	case DUPLICATE_SHARE:
		ret = "Duplicate share"
	case LOW_DIFFICULTY:
		ret = "LOW_DIFFICULTY"
	case NOT_LOGIN:
		ret = "Not login"
	case NOT_GETWORK:
		ret = "Not getwork"
	case ILLEGAL_PARARMS:
		ret = "Illegal params"
	case JOB_NOT_FOUND:
		ret = "Job not found"
	default:
		ret = "unknown"
	}

	return ret
}

func CompactToBig(compact uint32) *big.Int {
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

type Hash [HashSize]byte

func (hash *Hash) SetBytes(newHash []byte) error {
	nhlen := len(newHash)
	if nhlen != HashSize {
		return fmt.Errorf("invalid hash length of %v, want %v", nhlen,
			HashSize)
	}
	copy(hash[:], newHash)

	return nil
}

func HashToBig(hash *Hash) *big.Int {
	// A Hash is in little-endian, but the big package wants the bytes in
	// big-endian, so reverse them.
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

func reverseS(input string) (string, error) {
	a := strings.Split(input, "")
	sRev := ""
	//fmt.Println("a:", a, "  sRev:", sRev)
	if len(a)%2 != 0 {
		return "", fmt.Errorf("Incorrect input length")
	}
	for i := 0; i < len(a); i += 2 {
		tmp := []string{a[i], a[i+1], sRev}
		sRev = strings.Join(tmp, "")
	}
	return sRev, nil
}
