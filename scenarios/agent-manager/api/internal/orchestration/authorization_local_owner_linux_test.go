//go:build linux

package orchestration

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestCollectLocalPeerAncestryPinsInitialPeerIncarnation(t *testing.T) {
	bindings := map[int]cliutil.ProcessBinding{
		10: {PID: 10, UID: currentTestUID(), Start: "10", Boot: "boot", Cgroup: "/peer"},
		11: {PID: 11, UID: currentTestUID(), Start: "11", Boot: "boot", Cgroup: "/runner.scope"},
	}
	parents := map[int]int{10: 11, 11: 1}
	peerObservations := 0
	observe := func(pid int) (cliutil.ProcessBinding, error) {
		if pid == 10 {
			peerObservations++
			if peerObservations >= 2 {
				return cliutil.ProcessBinding{PID: 10, UID: currentTestUID(), Start: "replacement", Boot: "boot", Cgroup: "/peer"}, nil
			}
		}
		binding, ok := bindings[pid]
		if !ok {
			return cliutil.ProcessBinding{}, errors.New("process binding unavailable")
		}
		return binding, nil
	}
	if _, err := collectLocalPeerAncestry(context.Background(), 10, observe, fakeProcessStat(parents, map[int]string{10: "10", 11: "11"})); err == nil {
		t.Fatal("peer PID replacement between initial observation and ancestry walk was accepted")
	}
}

func TestCollectLocalPeerAncestryPinsObservedAncestorIncarnation(t *testing.T) {
	bindings := map[int]cliutil.ProcessBinding{
		10: {PID: 10, UID: currentTestUID(), Start: "10", Boot: "boot", Cgroup: "/peer"},
		11: {PID: 11, UID: currentTestUID(), Start: "11", Boot: "boot", Cgroup: "/runner.scope"},
	}
	parents := map[int]int{10: 11, 11: 1}
	ancestorObservations := 0
	observe := func(pid int) (cliutil.ProcessBinding, error) {
		if pid == 11 {
			ancestorObservations++
			if ancestorObservations >= 2 {
				return cliutil.ProcessBinding{PID: 11, UID: currentTestUID(), Start: "replacement", Boot: "boot", Cgroup: "/runner.scope"}, nil
			}
		}
		binding, ok := bindings[pid]
		if !ok {
			return cliutil.ProcessBinding{}, errors.New("process binding unavailable")
		}
		return binding, nil
	}
	if _, err := collectLocalPeerAncestry(context.Background(), 10, observe, fakeProcessStat(parents, map[int]string{10: "10", 11: "11"})); err == nil {
		t.Fatal("ancestor PID replacement between observations was accepted")
	}
}

func TestCollectLocalPeerAncestryTraversesUIDAndCgroupBoundaries(t *testing.T) {
	bindings := map[int]cliutil.ProcessBinding{
		10: {PID: 10, UID: currentTestUID(), Start: "10", Boot: "boot", Cgroup: "/peer"},
		11: {PID: 11, UID: currentTestUID() + 1, Start: "11", Boot: "boot", Cgroup: "/system.slice"},
		12: {PID: 12, UID: currentTestUID(), Start: "12", Boot: "boot", Cgroup: "/runner.scope"},
	}
	parents := map[int]int{10: 11, 11: 12, 12: 1}
	ancestry, err := collectLocalPeerAncestry(context.Background(), 10, fakeProcessObserver(bindings), fakeProcessStat(parents, map[int]string{10: "10", 11: "11", 12: "12"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(ancestry) != 3 || ancestry[0].binding.PID != 10 || ancestry[1].binding.PID != 11 || ancestry[2].binding.PID != 12 {
		t.Fatalf("ancestry stopped at an ownership boundary: %+v", ancestry)
	}
}

func TestCollectLocalPeerAncestryFailsClosedForMissingMalformedCycleAndDepth(t *testing.T) {
	t.Run("missing parent observation", func(t *testing.T) {
		bindings := map[int]cliutil.ProcessBinding{10: {PID: 10, UID: currentTestUID(), Start: "10", Boot: "boot", Cgroup: "/peer"}}
		parents := map[int]int{10: 11}
		if _, err := collectLocalPeerAncestry(context.Background(), 10, fakeProcessObserver(bindings), fakeProcessStat(parents, map[int]string{10: "10"})); err == nil {
			t.Fatal("missing parent was treated as a complete ancestry")
		}
	})

	t.Run("malformed stat", func(t *testing.T) {
		bindings := map[int]cliutil.ProcessBinding{10: {PID: 10, UID: currentTestUID(), Start: "10", Boot: "boot", Cgroup: "/peer"}}
		if _, err := collectLocalPeerAncestry(context.Background(), 10, fakeProcessObserver(bindings), func(int) ([]byte, error) { return []byte("not proc stat"), nil }); err == nil {
			t.Fatal("malformed ancestry stat was treated as absence")
		}
	})

	t.Run("cycle", func(t *testing.T) {
		bindings := map[int]cliutil.ProcessBinding{
			10: {PID: 10, UID: currentTestUID(), Start: "10", Boot: "boot", Cgroup: "/peer"},
			11: {PID: 11, UID: currentTestUID(), Start: "11", Boot: "boot", Cgroup: "/peer"},
		}
		parents := map[int]int{10: 11, 11: 10}
		if _, err := collectLocalPeerAncestry(context.Background(), 10, fakeProcessObserver(bindings), fakeProcessStat(parents, map[int]string{10: "10", 11: "11"})); err == nil {
			t.Fatal("cyclic ancestry was treated as complete")
		}
	})

	t.Run("depth limit", func(t *testing.T) {
		bindings := make(map[int]cliutil.ProcessBinding, maxLocalOwnerAncestry+1)
		parents := make(map[int]int, maxLocalOwnerAncestry+1)
		starts := make(map[int]string, maxLocalOwnerAncestry+1)
		for pid := 10; pid <= 10+maxLocalOwnerAncestry; pid++ {
			start := strconv.Itoa(pid)
			bindings[pid] = cliutil.ProcessBinding{PID: pid, UID: currentTestUID(), Start: start, Boot: "boot", Cgroup: "/peer"}
			parents[pid] = pid + 1
			starts[pid] = start
		}
		if _, err := collectLocalPeerAncestry(context.Background(), 10, fakeProcessObserver(bindings), fakeProcessStat(parents, starts)); err == nil {
			t.Fatal("ancestry beyond the bound was accepted")
		}
	})
}

func TestCollectLocalPeerAncestryRejectsReparentingDuringCheck(t *testing.T) {
	bindings := map[int]cliutil.ProcessBinding{
		10: {PID: 10, UID: currentTestUID(), Start: "10", Boot: "boot", Cgroup: "/peer"},
		11: {PID: 11, UID: currentTestUID(), Start: "11", Boot: "boot", Cgroup: "/peer"},
	}
	parents := map[int]int{10: 11, 11: 1}
	readCalls := map[int]int{}
	readStat := func(pid int) ([]byte, error) {
		readCalls[pid]++
		parent := parents[pid]
		if pid == 10 && readCalls[pid] > 1 {
			parent = 1
		}
		return syntheticProcessStat(parent, bindings[pid].Start), nil
	}
	if _, err := collectLocalPeerAncestry(context.Background(), 10, fakeProcessObserver(bindings), readStat); err == nil {
		t.Fatal("reparented peer ancestry was treated as stable")
	}
}

func currentTestUID() int { return os.Getuid() }

func fakeProcessObserver(bindings map[int]cliutil.ProcessBinding) func(int) (cliutil.ProcessBinding, error) {
	return func(pid int) (cliutil.ProcessBinding, error) {
		binding, ok := bindings[pid]
		if !ok {
			return cliutil.ProcessBinding{}, errors.New("process binding unavailable")
		}
		return binding, nil
	}
}

func fakeProcessStat(parents map[int]int, starts map[int]string) func(int) ([]byte, error) {
	return func(pid int) ([]byte, error) {
		parent, ok := parents[pid]
		if !ok {
			return nil, errors.New("process stat unavailable")
		}
		return syntheticProcessStat(parent, starts[pid]), nil
	}
}

func syntheticProcessStat(parent int, start string) []byte {
	fields := make([]string, 20)
	fields[0] = "S"
	fields[1] = strconv.Itoa(parent)
	for i := 2; i < len(fields); i++ {
		fields[i] = "0"
	}
	fields[19] = start
	return []byte("10 (test command) " + strings.Join(fields, " "))
}
