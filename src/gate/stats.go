package main

import (
	"sync/atomic"
	"time"
	"../apis/apps"
)

const (
	statsFlushPeriod	= 8
)

type FnStats struct {
//	ObjID		bson.ObjectId	`bson:"_id,omitempty"`
	Cookie		string		`bson:"cookie"`

	Called		uint64		`bson:"called"`
	LastCall	time.Time	`bson:"lastcall"`
	RunTime		time.Duration	`bson:"rtime"`
	CallTime	time.Duration	`bson:"ctime"`

	dirty		bool
	done		chan chan bool
	flushed		chan bool
}

type statsOpaque struct {
	ts		time.Time
}

func statsGet(fn *FunctionDesc) *FnStats {
	md := memdGetFn(fn)
	return &md.stats
}

func statsStart() *statsOpaque {
	return &statsOpaque{ts: time.Now()}
}

func statsUpdate(fmd *FnMemData, op *statsOpaque, res *swyapi.SwdFunctionRunResult) {
	fmd.stats.dirty = true
	atomic.AddUint64(&fmd.stats.Called, 1)
	fmd.stats.LastCall = op.ts
	fmd.stats.RunTime += time.Duration(res.Time) * time.Microsecond
	fmd.stats.CallTime += time.Since(op.ts)
}

var statsFlusher chan *FnStats

func statsInit(conf *YAMLConf) error {
	statsFlusher = make(chan *FnStats)
	go func() {
		for {
			st := <-statsFlusher
			dbStatsUpdate(st)
			st.flushed <- true
		}
	}()
	return nil
}

func statsDrop(fn *FunctionDesc) {
	md := memdGetCond(fn.Cookie)
	if md != nil {
		done := make(chan bool)
		md.stats.done <-done
		<-done

		dbStatsDrop(&md.stats)
	}
}

func fnStatsInit(st *FnStats, fn *FunctionDesc) {
	st.Cookie = fn.Cookie
	st.done = make(chan chan bool)
	st.flushed = make(chan bool)
	dbStatsGet(fn.Cookie, st)
	go func() {
		for {
			select {
			case done := <-st.done:
				done <- true
				return
			case <-time.After(statsFlushPeriod * time.Second):
				if st.dirty {
					st.dirty = false
					statsFlusher <-st
					<-st.flushed
				}
			}
		}
	}()
}
