package desktopwebrtc

import (
	"context"
	"testing"
	"time"
)

func TestBoundedQueueDropsFramesButProtectsControl(t *testing.T) {
	q := NewBoundedQueue[string](2)
	if _, err := q.Push("frame-1", true); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Push("frame-2", true); err != nil {
		t.Fatal(err)
	}
	dropped, err := q.Push("frame-3", true)
	if err != nil || !dropped || q.Len() != 2 {
		t.Fatalf("frame push dropped=%v err=%v len=%d", dropped, err, q.Len())
	}
	if _, err := q.Push("close", false); err == nil {
		t.Fatal("control should not displace queued frames")
	}
	q.Close()
	if _, err := q.Push("late", true); err != ErrQueueClosed {
		t.Fatalf("closed push error=%v", err)
	}
}

func TestBoundedQueuePopWaitWakesAndCloses(t *testing.T) {
	q := NewBoundedQueue[string](1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() { _, _ = q.Push("frame", true) }()
	value, ok := q.PopWait(ctx)
	if !ok || value != "frame" {
		t.Fatalf("value=%q ok=%v", value, ok)
	}
	q.Close()
	if _, ok := q.PopWait(ctx); ok {
		t.Fatal("closed queue returned an item")
	}
	var nilQueue *BoundedQueue[string]
	if _, err := nilQueue.Push("ignored", true); err != ErrQueueClosed {
		t.Fatalf("nil queue push error=%v", err)
	}
}
