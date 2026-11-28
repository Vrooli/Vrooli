package hub

import (
	"fmt"
	"testing"
	"time"
)

func TestParsePushBudgetConfig(t *testing.T) {
	got, err := ParsePushBudgetConfig("8", "30s")
	if err != nil || got != (PushBudgetConfig{Capacity: 8, RefillInterval: 30 * time.Second}) {
		t.Fatalf("ParsePushBudgetConfig = %+v, %v", got, err)
	}
	if got, err := ParsePushBudgetConfig("0", "ignored"); err != nil || got.Capacity != 0 {
		t.Fatalf("zero capacity should disable the budget: %+v, %v", got, err)
	}
	for _, values := range [][2]string{{"-1", "1m"}, {"x", "1m"}, {"1", "0s"}, {"1", "later"}} {
		if _, err := ParsePushBudgetConfig(values[0], values[1]); err == nil {
			t.Fatalf("ParsePushBudgetConfig%v accepted invalid input", values)
		}
	}
}

func TestPushBudgetRefillsContinuouslyAndIsPerSource(t *testing.T) {
	b := newPushBudget()
	b.config = PushBudgetConfig{Capacity: 2, RefillInterval: time.Minute}
	start := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	if !b.allowPush("run:a", start) || !b.allowPush("run:a", start) {
		t.Fatal("initial bucket should allow its full burst")
	}
	if b.allowPush("run:a", start) {
		t.Fatal("empty bucket allowed another device push")
	}
	if !b.allowPush("run:b", start) {
		t.Fatal("different sources must have independent buckets")
	}
	if b.allowPush("run:a", start.Add(30*time.Second)) {
		t.Fatal("a partial token must not enable a push")
	}
	if !b.allowPush("run:a", start.Add(time.Minute)) {
		t.Fatal("a full token should refill after one interval")
	}
}

func TestPushBudgetBoundsRetainedSources(t *testing.T) {
	b := newPushBudget()
	started := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 10_000; i++ {
		b.allowPush(fmt.Sprintf("source:%d", i), started)
	}
	b.allowPush("source:new", started.Add(time.Second))
	if len(b.buckets) != 10_000 {
		t.Fatalf("retained %d source buckets; want the configured memory ceiling of 10000", len(b.buckets))
	}
}

func TestDevicePushChannel(t *testing.T) {
	for _, channel := range []string{"web_push", "browser", "desktop", "macos_notification", "linux_notification"} {
		if !devicePushChannel(channel) {
			t.Errorf("%q should be suppressed when the push bucket is empty", channel)
		}
	}
	for _, channel := range []string{"email", "webhook", "telegram"} {
		if devicePushChannel(channel) {
			t.Errorf("%q should continue when only device pushes are budgeted", channel)
		}
	}
}
