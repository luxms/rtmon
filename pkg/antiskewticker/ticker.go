package antiskewticker

// please see https://github.com/golang/go/issues/19810 for details
import (
	"time"

	"github.com/amoghe/distillog"
)

// WallTicker keeps info about timer skew
type WallTicker struct {
	C      <-chan time.Time
	Trace  bool // will log trace info
	align  time.Duration
	offset time.Duration
	stop   chan bool
	c      chan time.Time
	skew   float64
	d      time.Duration
	last   time.Time
	tracer distillog.Logger
}

// NewWallTicker constructs new WallTicker struct
func NewWallTicker(align, offset time.Duration, trace bool, tracerIn distillog.Logger) *WallTicker {
	w := &WallTicker{
		Trace:  trace,
		align:  align,
		offset: offset,
		stop:   make(chan bool),
		c:      make(chan time.Time, 1),
		skew:   1.0,
		tracer: tracerIn,
	}
	w.C = w.c
	w.start()
	return w
}

// We probably know precise time skew for Azure
const fakeAzure = false

func (w *WallTicker) start() {
	now := time.Now()
	d := time.Until(now.Add(-w.offset).Add(w.align * 4 / 3).Truncate(w.align).Add(w.offset))
	d = time.Duration(float64(d) / w.skew)
	w.d = d
	w.last = now
	if fakeAzure {
		d = time.Duration(float64(d) * 99 / 101)
	}
	time.AfterFunc(d, w.tick)
}

func (w *WallTicker) tick() {
	const α = 0.7 // how much weight to give past history
	now := time.Now()
	if now.After(w.last) {
		w.skew = w.skew*α + (float64(now.Sub(w.last))/float64(w.d))*(1-α)
		select {
		case <-w.stop:
			if w.Trace {
				w.tracer.Infoln("ticker: stop")
			}
			return
		case w.c <- now:
			if w.Trace {
				w.tracer.Infof("ticker: calling func at %s\n", time.Now())
			}
			// ok
		default:
			// client not keeping up, drop tick
		}
	}
	w.start()
}

// example how to use this antiskew timer
/*
// The code below tries for 5s aligned + 0.01s offset
func main() {
	for range NewWallTicker(5*time.Second, 10*time.Millisecond).C {
		fmt.Println(time.Now())
	}
}
*/
