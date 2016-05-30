package null

import (
	"crypto/sha512"
	"encoding/base64"
	"io"
	"sort"

	"github.com/spacemonkeygo/errors"

	"polydawn.net/repeatr/api/def"
	"polydawn.net/repeatr/core/executor"
	"polydawn.net/repeatr/core/executor/basicjob"
	"polydawn.net/repeatr/core/model/formula"
	"polydawn.net/repeatr/lib/guid"
)

var _ executor.Executor = &Executor{}

type Mode int

const (
	Deterministic Mode = iota
	Nondeterministic
	SadExit
	Erroring
)

type Executor struct {
	Mode Mode
}

func (*Executor) Configure(workspacePath string) {
}

func (e *Executor) Start(f def.Formula, id executor.JobID, stdin io.Reader, journal io.Writer) executor.Job {
	job := basicjob.New(id)
	job.Result = executor.JobResult{
		ID:       id,
		ExitCode: -1,
	}

	go func() {
		switch e.Mode {
		case Deterministic:
			// seed hash with action and sorted input hashes.
			// (a real formula would behave a little differently around names v paths, but eh.)
			// ... actually this is basically the same as the stage2 identity.  yeah.  tis.
			// which arguably makes it kind of a dangerously cyclic for some tests.  but there's
			// not much to be done about that aside from using real executors in your tests then.
			hasher := sha512.New384()
			hasher.Write([]byte(formula.Stage2(f).ID()))

			// aside: fuck you golang for making me write this goddamned sort AGAIN.
			// my kingdom for a goddamn sorted map.
			keys := make([]string, len(f.Outputs))
			var i int
			for k := range f.Outputs {
				keys[i] = k
				i++
			}
			sort.Strings(keys)

			// emit outputs, using their names in sorted order to predictably advance
			//  the hash state, while drawing their ids back.
			job.Result.Outputs = def.OutputGroup{}
			for _, name := range keys {
				hasher.Write([]byte(name))
				job.Result.Outputs[name] = f.Outputs[name].Clone()
				job.Result.Outputs[name].Hash = base64.URLEncoding.EncodeToString(hasher.Sum(nil))
			}
		case Nondeterministic:
			job.Result.Outputs = def.OutputGroup{}
			for name, spec := range f.Outputs {
				job.Result.Outputs[name] = spec.Clone()
				job.Result.Outputs[name].Hash = guid.New()
			}
		case SadExit:
			job.Result.ExitCode = 4
		case Erroring:
			job.Result.Error = executor.TaskExecError.New("mock error").(*errors.Error)
		default:
			panic("no")
		}

		close(job.WaitChan)
	}()

	return job
}
