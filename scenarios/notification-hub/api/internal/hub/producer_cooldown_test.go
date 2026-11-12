package hub

import (
	"errors"
	"testing"
	"time"
)

func TestParseCooldownLimit(t *testing.T) {
	for value, want := range map[string]CooldownLimit{
		"6/10m": {Count: 6, Window: 10 * time.Minute},
		" 3/1h": {Count: 3, Window: time.Hour},
		"off":   {},
		"0":     {},
	} {
		got, err := ParseCooldownLimit(value)
		if err != nil || got != want {
			t.Fatalf("ParseCooldownLimit(%q) = %+v, %v; want %+v", value, got, err, want)
		}
	}
	for _, value := range []string{"6", "0/10m", "-1/10m", "6/soon", "6/0s", ""} {
		if _, err := ParseCooldownLimit(value); err == nil {
			t.Fatalf("ParseCooldownLimit(%q) accepted an invalid limit", value)
		}
	}
}

// The window slides: a slot frees when its oldest admission ages out, a
// released slot is returned, and kinds and producers never share budgets.
func TestProducerCooldownSlidingWindow(t *testing.T) {
	c := newProducerCooldown()
	c.limits[ProducerNotification] = CooldownLimit{Count: 2, Window: 10 * time.Minute}
	c.limits[ProducerAsk] = CooldownLimit{Count: 1, Window: 10 * time.Minute}
	start := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)

	if _, err := c.admit("run:a", ProducerNotification, start); err != nil {
		t.Fatal(err)
	}
	release, err := c.admit("run:a", ProducerNotification, start.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.admit("run:a", ProducerNotification, start.Add(3*time.Minute))
	var refusal *CooldownError
	if !errors.As(err, &refusal) || !errors.Is(err, ErrProducerCooldown) || !refusal.RetryAt.Equal(start.Add(10*time.Minute)) || refusal.RetryAfter != 7*time.Minute {
		t.Fatalf("third send = %v; want a refusal until 12:10", err)
	}
	release()
	if _, err := c.admit("run:a", ProducerNotification, start.Add(3*time.Minute)); err != nil {
		t.Fatalf("a released slot must be reusable: %v", err)
	}
	if _, err := c.admit("run:a", ProducerAsk, start.Add(3*time.Minute)); err != nil {
		t.Fatalf("asks have their own budget: %v", err)
	}
	if _, err := c.admit("run:b", ProducerNotification, start.Add(3*time.Minute)); err != nil {
		t.Fatalf("each producer has its own budget: %v", err)
	}
	if _, err := c.admit("run:a", ProducerNotification, start.Add(10*time.Minute)); err != nil {
		t.Fatalf("the oldest admission ages out after the window: %v", err)
	}
	c.limits[ProducerNotification] = CooldownLimit{}
	for i := 0; i < 20; i++ {
		if _, err := c.admit("run:a", ProducerNotification, start.Add(11*time.Minute)); err != nil {
			t.Fatalf("a disabled limit admits every send: %v", err)
		}
	}
}
