package main

import (
	"fmt"
	"context"
	"sync"
	"time"
)

func condWaitTmo(cond *sync.Cond, tmo time.Duration) {
	d := time.AfterFunc(tmo, func() { cond.Signal() })
	cond.Wait()
	d.Stop()
}

func balancerFnScaler(fdm *FnMemData) {
up:
	glog.Debugf("Scale %s up to %d", fdm.depname, fdm.bd.goal)
	goal := swk8sDepScaleUp(fdm.depname, fdm.bd.goal)

	fdm.lock.Lock()
	if fdm.bd.goal == 0 {
		goto fin
	}

	if fdm.bd.goal > goal {
		fdm.lock.Unlock()
		goto up
	}
relax:
	condWaitTmo(fdm.bd.wakeup, SwyDepScaleupRelax)

down:
	if fdm.bd.goal <= 1 {
		fdm.bd.wakeup = nil
		goto fin
	}
	if fdm.bd.goal > goal {
		fdm.lock.Unlock()
		goto up
	}

	fdm.bd.goal--
	condWaitTmo(fdm.bd.wakeup, SwyDepScaledownStep)
	if fdm.bd.goal == 0 {
		goto fin
	}
	if fdm.bd.goal == goal {
		goto relax
	}
	if fdm.bd.goal > goal {
		fdm.lock.Unlock()
		goto up
	}

	fdm.lock.Unlock()
	glog.Debugf("Scale %s down to %d", fdm.depname, fdm.bd.goal)
	goal = swk8sDepScaleDown(fdm.depname, fdm.bd.goal)
	fdm.lock.Lock()

	goto down

fin:
	fdm.lock.Unlock()
	glog.Debugf("Scaler %s done", fdm.depname)
}

func balancerFnDepGrow(ctx context.Context, fdm *FnMemData, goal uint32) {
	if goal <= fdm.bd.goal {
		return
	}

	fdm.lock.Lock()
	if goal <= fdm.bd.goal {
		fdm.lock.Unlock()
		return
	}

	if goal > conf.Runtime.MaxReplicas {
		fdm.lock.Unlock()
		ctxlog(ctx).Debugf("Too many replicas (%d) needed for %s", goal, fdm.depname)
		return
	}

	fdm.bd.goal = goal

	if fdm.bd.wakeup == nil {
		fdm.bd.wakeup = sync.NewCond(&fdm.lock)
		go balancerFnScaler(fdm)
	} else {
		fdm.bd.wakeup.Signal()
	}
	fdm.lock.Unlock()
}

func scalerInit(fn *FunctionDesc, tgt uint32) error {
	fdm, err := memdGetFn(fn)
	if err != nil {
		return fmt.Errorf("Can't get fdmd for %s: %s", fn.SwoId.Str(), err.Error())
	}

	fdm.bd.goal = tgt
	fdm.bd.wakeup = sync.NewCond(&fdm.lock)
	go balancerFnScaler(fdm)

	return nil
}
