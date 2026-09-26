package store

import (
	"testing"
	"time"
)

func TestEffectiveTimeout_Default(t *testing.T) {
	c := &HeartbeatConfig{TimeoutSeconds: 0}
	got := c.EffectiveTimeout()
	want := 45 * time.Minute
	if got != want {
		t.Errorf("EffectiveTimeout() = %v, want %v", got, want)
	}
}

func TestEffectiveTimeout_Custom(t *testing.T) {
	c := &HeartbeatConfig{TimeoutSeconds: 3600}
	got := c.EffectiveTimeout()
	want := 1 * time.Hour
	if got != want {
		t.Errorf("EffectiveTimeout() = %v, want %v", got, want)
	}
}

func TestEffectiveTimeout_ClampMin(t *testing.T) {
	c := &HeartbeatConfig{TimeoutSeconds: 10}
	got := c.EffectiveTimeout()
	want := 60 * time.Second
	if got != want {
		t.Errorf("EffectiveTimeout() = %v, want %v (should clamp to min)", got, want)
	}
}

func TestEffectiveTimeout_ClampMax(t *testing.T) {
	c := &HeartbeatConfig{TimeoutSeconds: 99999}
	got := c.EffectiveTimeout()
	want := 7200 * time.Second
	if got != want {
		t.Errorf("EffectiveTimeout() = %v, want %v (should clamp to max)", got, want)
	}
}

func TestEffectiveTimeout_Negative(t *testing.T) {
	c := &HeartbeatConfig{TimeoutSeconds: -100}
	got := c.EffectiveTimeout()
	want := 45 * time.Minute
	if got != want {
		t.Errorf("EffectiveTimeout() = %v, want %v (negative should use default)", got, want)
	}
}

func TestEffectiveTimeout_ExactMin(t *testing.T) {
	c := &HeartbeatConfig{TimeoutSeconds: MinHeartbeatTimeoutSeconds}
	got := c.EffectiveTimeout()
	want := time.Duration(MinHeartbeatTimeoutSeconds) * time.Second
	if got != want {
		t.Errorf("EffectiveTimeout() = %v, want %v", got, want)
	}
}

func TestEffectiveTimeout_ExactMax(t *testing.T) {
	c := &HeartbeatConfig{TimeoutSeconds: MaxHeartbeatTimeoutSeconds}
	got := c.EffectiveTimeout()
	want := time.Duration(MaxHeartbeatTimeoutSeconds) * time.Second
	if got != want {
		t.Errorf("EffectiveTimeout() = %v, want %v", got, want)
	}
}
