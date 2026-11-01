package androidadb

import (
	"context"
	"device-control/strategy"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var surfaceIdentity = regexp.MustCompile(`^SurfaceView - [A-Za-z0-9_.$]+/[A-Za-z0-9_.$]+#[0-9]+$`)

func parseGraphicsPerformance(data string, p *strategy.AppPerformance) {
	fields := []struct {
		pattern string
		dst     **int64
	}{
		{`(?m)^\s*Stats since:\s*(\d+)ns`, &p.GraphicsSinceNS},
		{`(?m)^\s*Total frames rendered:\s*(\d+)`, &p.GraphicsFrames},
		{`(?m)^\s*Janky frames:\s*(\d+)`, &p.JankyFrames},
		{`(?m)^\s*Number Missed Vsync:\s*(\d+)`, &p.MissedVsync},
	}
	for _, field := range fields {
		if m := regexp.MustCompile(field.pattern).FindStringSubmatch(data); len(m) == 2 {
			if n, e := strconv.ParseInt(m[1], 10, 64); e == nil {
				*field.dst = &n
			}
		}
	}
}

func parseSurfacePerformance(data string, uptime float64, p *strategy.AppPerformance) {
	lines := strings.Split(strings.TrimSpace(data), "\n")
	if len(lines) < 2 {
		p.SurfaceReason = "presentation data unavailable"
		return
	}
	refresh, err := strconv.ParseInt(strings.TrimSpace(lines[0]), 10, 64)
	if err != nil || refresh <= 0 {
		p.SurfaceReason = "display refresh period unavailable"
		return
	}
	var stamps []int64
	for _, line := range lines[1:] {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		n, e := strconv.ParseInt(f[1], 10, 64)
		if e == nil && n > 0 && n < math.MaxInt64 {
			stamps = append(stamps, n)
		}
	}
	sort.Slice(stamps, func(i, j int) bool { return stamps[i] < stamps[j] })
	unique := stamps[:0]
	for _, s := range stamps {
		if len(unique) == 0 || s != unique[len(unique)-1] {
			unique = append(unique, s)
		}
	}
	stamps = unique
	if len(stamps) > 0 {
		cutoff := stamps[len(stamps)-1] - int64(5*time.Second)
		first := sort.Search(len(stamps), func(i int) bool { return stamps[i] >= cutoff })
		stamps = stamps[first:]
	}
	p.SurfaceFrames = len(stamps)
	if len(stamps) < 2 {
		p.SurfaceReason = "insufficient presentation timestamps"
		return
	}
	age := uptime*1000 - float64(stamps[len(stamps)-1])/1e6
	if age < 0 {
		p.SurfaceReason = "presentation clock does not match device uptime"
		return
	}
	p.SurfaceAgeMS = &age
	p.SurfaceWindowMS = float64(stamps[len(stamps)-1]-stamps[0]) / 1e6
	if age > 2000 {
		p.SurfaceReason = "presentation timestamps are stale"
		return
	}
	fps := float64(len(stamps)-1) * 1e9 / float64(stamps[len(stamps)-1]-stamps[0])
	p.SurfaceFPS = &fps
	gaps := make([]float64, 0, len(stamps)-1)
	for i := 1; i < len(stamps); i++ {
		d := stamps[i] - stamps[i-1]
		gaps = append(gaps, float64(d)/1e6)
		if d > refresh*3/2 {
			p.SlowIntervals++
		}
	}
	sort.Float64s(gaps)
	p95 := gaps[int(math.Ceil(float64(len(gaps))*0.95))-1]
	p.SurfaceP95MS = &p95
}

func (a *Adapter) performanceState(ctx context.Context, pkg string) (strategy.AppLifecycleResult, error) {
	r := strategy.AppLifecycleResult{Operation: "performance-state", Package: pkg, Status: strategy.StatusUnavailable}
	bounded, ok := a.runner.(boundedRunner)
	if !ok {
		return r, fmt.Errorf("bounded performance diagnostics unavailable")
	}
	run := func(args ...string) ([]byte, error) {
		return bounded.RunBounded(ctx, "adb", 256*1024, a.args(args...)...)
	}
	p := &strategy.AppPerformance{}
	if data, err := run("shell", "dumpsys", "gfxinfo", pkg); err == nil {
		parseGraphicsPerformance(string(data), p)
	}
	data, err := run("shell", "dumpsys", "SurfaceFlinger", "--list")
	var matches []string
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "SurfaceView - "+pkg+"/") && surfaceIdentity.MatchString(line) {
				matches = append(matches, line)
			}
		}
	}
	switch len(matches) {
	case 0:
		p.SurfaceReason = "no matching video surface"
	case 1:
		latency, e := run("shell", "dumpsys", "SurfaceFlinger", "--latency", "'"+matches[0]+"'")
		uptime, u := run("shell", "cat", "/proc/uptime")
		if e != nil || u != nil {
			p.SurfaceReason = "presentation data unavailable"
		} else {
			fields := strings.Fields(string(uptime))
			if len(fields) == 0 {
				p.SurfaceReason = "device uptime unavailable"
			} else if n, e := strconv.ParseFloat(fields[0], 64); e == nil {
				parseSurfacePerformance(string(latency), n, p)
			} else {
				p.SurfaceReason = "device uptime unavailable"
			}
		}
	default:
		p.SurfaceReason = "multiple matching video surfaces"
	}
	if ctx.Err() != nil {
		return r, ctx.Err()
	}
	if p.GraphicsFrames == nil && p.SurfaceFPS == nil {
		return r, fmt.Errorf("performance metrics unavailable: %s", p.SurfaceReason)
	}
	r.Performance = p
	r.Status = strategy.StatusAvailable
	r.Outcome = "performance_observed"
	r.ObservedAt = time.Now().UTC()
	return r, nil
}
