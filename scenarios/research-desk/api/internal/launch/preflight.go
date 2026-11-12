package launch

import (
	"time"

	"research-desk/internal/home"
)

// Resolution is the operator's choice for an earlier commission that blocks
// a launch (rulings P2).
type Resolution string

// Resolutions.
const (
	// Supersede stops the earlier supervisor (only through its own thread and
	// session) and records it as superseded by the new commission.
	Supersede Resolution = "supersede"
	// RunAlongside lets both run.
	RunAlongside Resolution = "run_alongside"
)

// QuietAfter is how long an unknown commission must have gone without a
// write, and without an open session, to be listed as quiet rather than
// block a launch (rulings P2).
const QuietAfter = 24 * time.Hour

// Verdict says what one earlier commission on a topic means for a launch.
type Verdict struct {
	Topic        string
	CommissionID string
	State        home.State
	Blocks       bool
	Message      string
	// ReadFirst is the newest checkpoint of a stopped or ended commission on
	// the topic, offered as the spec's read_first.
	ReadFirst string
	// Quiet is an unknown commission with no open session and no write in
	// QuietAfter: listed, never blocking.
	Quiet bool
	// Choices clear a blocker; Recommended is the desk's choice, empty when
	// the operator must choose; Chosen is the request's choice.
	Choices           []Resolution
	Recommended       Resolution
	RecommendedReason string
	Chosen            Resolution
}

// Verdicts covers every commission other than self that names one of the
// topics (SYSTEM.md §5). A live, paused or limited commission blocks, and so
// does an unknown one with an open session or a recent write, unless the
// choices resolve it.
func Verdicts(snap *home.Snapshot, topics []string, choices map[string]Resolution, self string) []Verdict {
	var out []Verdict
	for _, topic := range topics {
		newest := -1
		var newestAt int64
		for _, c := range snap.Commissions {
			if c.ID == self || !names(c, topic) {
				continue
			}
			v := Verdict{Topic: topic, CommissionID: c.ID, State: c.State}
			switch c.State {
			case home.StateLive, home.StatePaused, home.StateLimited, home.StateUnknown:
				v.judgeOpen(c, snap.Now, choices[c.ID])
			case home.StatePrepared:
				v.Message = "another unlaunched packet covers this topic; discard it or launch only one"
			default:
				v.Message = "nothing to do (" + string(c.State) + ")"
				if cp := checkpoint(c, topic); cp != nil && cp.Modified.UnixNano() > newestAt {
					newest, newestAt = len(out), cp.Modified.UnixNano()
				}
			}
			out = append(out, v)
		}
		if newest >= 0 {
			v := &out[newest]
			v.ReadFirst = checkpoint(snap.Commission(v.CommissionID), topic).Path
			v.Message += "; its newest checkpoint is offered as read_first"
		}
	}
	return out
}

// judgeOpen fills the verdict for a commission that may still be running.
func (v *Verdict) judgeOpen(c *home.Commission, now time.Time, chosen Resolution) {
	if c.State == home.StateUnknown {
		if quiet, why := quietUnknown(c, now); quiet {
			v.Quiet, v.Message = true, "state unknown ("+c.StateRule+") but quiet: "+why+"; listed, not blocking"
			return
		} else {
			v.Message = "state unknown (" + c.StateRule + ") and " + why + ", so it may still be running"
		}
	} else {
		v.Message = string(c.State)
	}
	v.Choices = []Resolution{Supersede, RunAlongside}
	v.Recommended, v.RecommendedReason = recommend(c, now)
	switch chosen {
	case RunAlongside:
		v.Chosen, v.Message = chosen, v.Message+"; runs alongside this launch by the operator's choice"
	case Supersede:
		v.Chosen = chosen
		if c.State.Running() {
			v.Message += "; superseded by this launch: its supervisor is stopped first"
		} else {
			v.Message += "; superseded by this launch: recorded, no supervisor window to stop"
		}
	default:
		v.Blocks, v.Message = true, v.Message+"; choose supersede or run alongside"
	}
}

// quietUnknown says whether an unknown commission has no open session and
// no write in QuietAfter, and why or why not.
func quietUnknown(c *home.Commission, now time.Time) (bool, string) {
	switch {
	case c.Session != nil && c.Session.WindowOpen:
		return false, "its session " + c.Session.SessionID + " is open"
	case c.Session != nil:
		return false, "its session " + c.Session.SessionID + " may be open (no window check)"
	case !c.LastWrite.Known:
		return false, "its last write is unknown (" + c.LastWrite.Reason + ")"
	case now.Sub(c.LastWrite.Value) < QuietAfter:
		return false, "it was written at " + stamp(c.LastWrite.Value) + ", within 24 hours"
	}
	return true, "no open session, last write " + stamp(c.LastWrite.Value)
}

// recommend is the desk's choice for a blocker: supersede a paused or
// limited commission whose allowance has ended; nothing for a live one or
// when the desk cannot tell.
func recommend(c *home.Commission, now time.Time) (Resolution, string) {
	switch {
	case c.State == home.StateLive:
		return "", "it is live: the operator chooses"
	case (c.State == home.StatePaused || c.State == home.StateLimited) && c.AllowanceEnd.Known && now.After(c.AllowanceEnd.Value):
		return Supersede, string(c.State) + " and its allowance ended at " + stamp(c.AllowanceEnd.Value)
	case c.State == home.StatePaused || c.State == home.StateLimited:
		return "", string(c.State) + " with allowance left: the operator chooses"
	}
	return "", "its state is unknown: the operator chooses"
}

func stamp(t time.Time) string { return t.UTC().Format("2006-01-02 15:04Z") }

func names(c *home.Commission, topic string) bool {
	for _, t := range c.Topics {
		if t.Topic == topic {
			return true
		}
	}
	return false
}

func checkpoint(c *home.Commission, topic string) *home.FileFact {
	for _, r := range c.Runs {
		if r.Topic == topic && r.Checkpoint != nil {
			return r.Checkpoint
		}
	}
	return nil
}
