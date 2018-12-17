package timing

import (
	"time"
)

// Timing handles timing things.
type Timing struct {
	StartedAt time.Time
	StoppedAt time.Time
}

// Start returns an instance of Timing with the Start field timestamped.
func Start() Timing {
	return Timing{
		StartedAt: time.Now(),
	}
}

// Start records the start time of Now.
func (t *Timing) Start() {
	t.StartedAt = time.Now()
}

// Stop records the stop time of Now.
func (t *Timing) Stop() {
	t.StoppedAt = time.Now()
}

// Elapsed returns the duration from Timing.StartedAt until Timing.StoppedAt.
func (t *Timing) Elapsed() time.Duration {
	return t.StoppedAt.Sub(t.StartedAt)
}

// ElapsedString returns the string of Elapsed
func (t *Timing) ElapsedString() string {
	return t.Elapsed().String()
}
