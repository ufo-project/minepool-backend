package main

import (
	"time"
)

const (
	rateLow  = 0.4
	rateHigh = 1.4
)

type diffController struct {
	startTime time.Time
	curDiff   float64

	lastRingIdx uint
	elements    []uint
}

func newDiffController(diff float64) *diffController {
	return &diffController{
		startTime: time.Now(),
		curDiff:   diff,

		lastRingIdx: 0,
		elements:    make([]uint, cfg.WindowSize),
	}
}

func (dc *diffController) addShare() {
	curRingIdx := uint(time.Now().Unix()) / cfg.SecondsPerShare

	if dc.lastRingIdx == 0  || curRingIdx-dc.lastRingIdx > cfg.WindowSize {
		dc.Clear()
		dc.lastRingIdx = curRingIdx
	}

	for dc.lastRingIdx < curRingIdx {
		dc.lastRingIdx++
		dc.elements[dc.lastRingIdx%cfg.WindowSize] = 0
	}

	dc.elements[curRingIdx%cfg.WindowSize] += 1
}

func (dc *diffController) calcCurDiff() float64 {
	timestamp := uint(time.Now().Unix())
	index := timestamp / cfg.SecondsPerShare

	sharesNum := dc.Sum(index)
	expectedNum := cfg.WindowSize

	if float64(sharesNum) > float64(expectedNum)*rateHigh {
		for {
			diff := dc.curDiff * 2
			if sharesNum > expectedNum && diff < cfg.MaxDiff {
				dc.Divide(index, 2)
				dc.curDiff = diff

				sharesNum = dc.Sum(index)
			}
			return dc.curDiff
		}
	}

	if float64(sharesNum) < float64(expectedNum)*rateLow && dc.isFullWindow(timestamp) {
		for {
			diff := dc.curDiff / 2
			if sharesNum < expectedNum && diff > cfg.MinDiff {
				dc.Multiply(index, 2)
				dc.curDiff = diff
				sharesNum = dc.Sum(index)
			}
			return dc.curDiff
		}
	}

	return dc.curDiff
}

func (dc *diffController) isFullWindow(timestamp uint) bool {
	return timestamp >= uint(dc.startTime.Unix())+cfg.TotalSeconds
}

func (dc *diffController) Clear() {
	for i := range dc.elements {
		dc.elements[i] = 0
	}
}

func (dc *diffController) Sum(curRingIdx uint) uint {
	var sum uint

	if curRingIdx-cfg.WindowSize >= dc.lastRingIdx {
		return 0
	}

	startRingIdx := curRingIdx - cfg.WindowSize
	endRingIdx := dc.lastRingIdx

	for endRingIdx > startRingIdx {
		i := endRingIdx % cfg.WindowSize
		sum += dc.elements[i]
		endRingIdx--
	}
	return sum
}

func (dc *diffController) Divide(curRingIdx, divisor uint) {

	if curRingIdx-cfg.WindowSize >= dc.lastRingIdx {
		return
	}

	startRingIdx := curRingIdx - cfg.WindowSize
	endRingIdx := dc.lastRingIdx

	for endRingIdx > startRingIdx {
		i := endRingIdx % cfg.WindowSize
		dc.elements[i] /= divisor
		endRingIdx--
	}
}

func (dc *diffController) Multiply(curRingIdx, multiplier uint) {

	if curRingIdx-cfg.WindowSize >= dc.lastRingIdx {
		return
	}

	startRingIdx := curRingIdx - cfg.WindowSize
	endRingIdx := dc.lastRingIdx

	for endRingIdx > startRingIdx {
		i := endRingIdx % cfg.WindowSize
		dc.elements[i] *= multiplier
		endRingIdx--
	}
}
