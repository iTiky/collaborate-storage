package server

import (
	"log"
	"sync"
	"time"

	movingaverage "github.com/RobinUS2/golang-moving-average"
)

var monitor *Monitor

// Monitor keeps SortedListService stats.
type Monitor struct {
	sync.Mutex
	opsHandled  int
	diffHandled int
	diffReqDur  *movingaverage.MovingAverage
	stopCh      chan struct{}
}

// OpsHandled increments the storage operations performed metric.
func (m *Monitor) OpsHandled(count int) {
	m.Lock()
	defer m.Unlock()

	m.opsHandled += count
}

// DiffRequestServed updates the service.GetListUpdates handling duration metric.
func (m *Monitor) DiffRequestServed(dur time.Duration) {
	m.Lock()
	defer m.Unlock()

	m.diffReqDur.Add(float64(dur/time.Microsecond) / 1000.0)
	m.diffHandled++
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

			updsPerSec := float64(m.opsHandled) / (float64(period) / float64(time.Second))
			diffsPerSec := float64(m.diffHandled) / (float64(period) / float64(time.Second))
			log.Printf("Monitor:")
			log.Printf("  - Storate updates / s:   %.2f", updsPerSec)
			log.Printf("  - Diff requests / s:     %.2f", diffsPerSec)
			log.Printf("  - Diff request dur [ms]: %.2f", m.diffReqDur.Avg())
			m.opsHandled = 0
			m.diffHandled = 0

			m.Unlock()
		}
	}
}

func init() {
	monitor = &Monitor{
		diffReqDur: movingaverage.New(5),
	}
}
