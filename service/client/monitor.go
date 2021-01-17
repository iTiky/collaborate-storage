package client

import (
	"log"
	"sync"
	"time"

	movingaverage "github.com/RobinUS2/golang-moving-average"
)

var monitor *Monitor

// Monitor keeps Client stats.
type Monitor struct {
	sync.Mutex
	updReqDur        *movingaverage.MovingAverage
	diffReqDur       *movingaverage.MovingAverage
	consistencyDur   *movingaverage.MovingAverage
	updReqSend       int
	diffReqSend      int
	consistencyReset time.Time
	stopCh           chan struct{}
}

func (m *Monitor) UpdatesSend(count int, dur time.Duration) {
	m.Lock()
	defer m.Unlock()

	m.updReqSend += count
	m.updReqDur.Add(float64(dur/time.Microsecond) / 1000.0)
}

func (m *Monitor) UpdatesReceived(count int, dur time.Duration) {
	m.Lock()
	defer m.Unlock()

	m.diffReqSend += count
	m.diffReqDur.Add(float64(dur/time.Microsecond) / 1000.0)
}

func (m *Monitor) ConsistencyReset(ts time.Time) {
	m.Lock()
	defer m.Unlock()

	m.consistencyReset = ts
}

func (m *Monitor) ConsistencyAchieved(ts time.Time) {
	m.Lock()
	defer m.Unlock()

	dur := ts.Sub(m.consistencyReset)
	m.consistencyDur.Add(float64(dur/time.Microsecond) / 1000.0)
}

// Start starts the Monitor worker.
func (m *Monitor) Start() {
	if m.stopCh != nil {
		return
	}

	m.stopCh = make(chan struct{})
	go m.worker()
}

// Stop stops the Monitor worker.
func (m *Monitor) Stop() {
	if m.stopCh == nil {
		return
	}

	close(m.stopCh)
}

// worker does the actual job.
func (m *Monitor) worker() {
	const period = 5 * time.Second

	tickCh := time.Tick(period)
	for {
		select {
		case <-m.stopCh:
			// Stop the monitor
			return
		case <-tickCh:
			// Print the report
			m.Lock()

			updReqPerSec := float64(m.updReqSend) / (float64(period) / float64(time.Second))
			diffReqPerSec := float64(m.diffReqSend) / (float64(period) / float64(time.Second))
			log.Printf("Monitor:")
			log.Printf("  - Update requests / s:     %.2f", updReqPerSec)
			log.Printf("  - Diff requests / s:       %.2f", diffReqPerSec)
			log.Printf("  - Update request dur [ms]: %.2f", m.updReqDur.Avg())
			log.Printf("  - Diff request dur [ms]:   %.2f", m.diffReqDur.Avg())
			log.Printf("  - Consistancy dur [ms]:    %.2f", m.consistencyDur.Avg())
			m.updReqSend = 0
			m.diffReqSend = 0

			m.Unlock()
		}
	}
}

func init() {
	monitor = &Monitor{
		updReqDur:      movingaverage.New(3),
		diffReqDur:     movingaverage.New(3),
		consistencyDur: movingaverage.New(3),
	}
}
