package sender

import (
	"math/rand"
	"time"
)

type Backoff struct {
	Initial    time.Duration
	Max        time.Duration
	Multiplier float64
	JitterFrac float64
	MaxElapsed time.Duration

	current time.Duration
	started time.Time
}

func NewBackoff(maxElapsed time.Duration) *Backoff {
	return &Backoff{
		Initial:    2 * time.Second,
		Max:        60 * time.Second,
		Multiplier: 2.0,
		JitterFrac: 0.25,
		MaxElapsed: maxElapsed,
	}
}

func (b *Backoff) Reset() {
	b.current = b.Initial
	b.started = time.Now()
}

// NextDelay returns the next sleep duration and true, or 0/false if MaxElapsed exceeded.
func (b *Backoff) NextDelay() (time.Duration, bool) {
	if b.current == 0 {
		b.Reset()
	}
	if time.Since(b.started)+b.current > b.MaxElapsed {
		return 0, false
	}
	d := b.current
	jitter := time.Duration(float64(d) * b.JitterFrac * (2*rand.Float64() - 1))
	d += jitter
	if d < 0 {
		d = b.Initial
	}
	next := time.Duration(float64(b.current) * b.Multiplier)
	if next > b.Max {
		next = b.Max
	}
	b.current = next
	return d, true
}
