package tests

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"go.polydawn.net/go-timeless-api"
	"go.polydawn.net/go-timeless-api/repeatr"
	. "go.polydawn.net/repeatr/testutil"
)

func shouldRun(t *testing.T, runTool repeatr.RunFunc, frm api.Formula, frmCtx api.FormulaContext) (api.RunRecord, string) {
	rr, txt, err := run(t, runTool, frm, baseFormulaCtx)
	AssertNoError(t, err)
	return *rr, txt
}
func run(t *testing.T, runTool repeatr.RunFunc, frm api.Formula, frmCtx api.FormulaContext) (*api.RunRecord, string, error) {
	bm := bufferingMonitor{}
	rr, err := runTool(context.Background(), frm, baseFormulaCtx, repeatr.InputControl{}, bm.monitor())
	bm.await()
	return rr, bm.Txt.String(), err
}

type bufferingMonitor struct {
	Ch  chan repeatr.Event
	Wg  sync.WaitGroup
	Txt bytes.Buffer
	Err error
}

func (bm *bufferingMonitor) monitor() repeatr.Monitor {
	*bm = bufferingMonitor{
		Ch: make(chan repeatr.Event),
	}
	bm.Wg.Add(1)
	go func() {
		defer bm.Wg.Done()
		for msg := range bm.Ch {
			bm.Err = repeatr.CopyOut(msg, &bm.Txt)
		}
	}()
	return repeatr.Monitor{Chan: bm.Ch}
}

func (bm *bufferingMonitor) await() error {
	bm.Wg.Wait()
	return bm.Err
}
