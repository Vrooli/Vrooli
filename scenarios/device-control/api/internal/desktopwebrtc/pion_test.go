package desktopwebrtc

import (
	"context"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
)

func TestPionPeerCreatesVP8AndReliableLanes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	peer, err := NewPionPeer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if peer.video == nil || peer.control == nil || peer.data == nil {
		t.Fatalf("peer lanes not initialized: %+v", peer)
	}
	if err := peer.AddVP8Sample(ctx, []byte{0x10, 0x00, 0x00}, 0); err != nil {
		t.Fatal(err)
	}
	if err := peer.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPionPeerReportsBoundedTransportCounters(t *testing.T) {
	peer, err := NewPionPeer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	peer.controlQueue.Close()
	peer.dataQueue.Close()
	if err := peer.WriteControl([]byte("control")); err == nil {
		t.Fatal("closed control lane accepted a message")
	}
	if err := peer.WriteData([]byte("data")); err == nil {
		t.Fatal("closed data lane accepted a message")
	}
	stats := peer.Stats()
	if stats.GetControlMessagesRejected() != 1 || stats.GetDataMessagesRejected() != 1 {
		t.Fatalf("stats=%v", stats)
	}
}

func TestPionPeerNotifiesOwnerWhenBrowserClosesPeer(t *testing.T) {
	peer, err := NewPionPeer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	failed := make(chan struct{})
	peer.SetFailureHandler(func() { close(failed) })
	if err := peer.pc.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-failed:
	case <-time.After(time.Second):
		t.Fatal("peer close did not notify the owner")
	}
}

func TestPionPeerAnswersBrowserOffer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	companion, err := NewPionPeer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer companion.Close()

	api := webrtc.NewAPI()
	browser, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer browser.Close()
	if _, err := browser.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		t.Fatal(err)
	}
	offer, err := browser.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := browser.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	select {
	case <-webrtc.GatheringCompletePromise(browser):
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	local := browser.LocalDescription()
	if local == nil {
		t.Fatal("browser offer was not gathered")
	}
	answer, err := companion.Answer(ctx, *local)
	if err != nil {
		t.Fatal(err)
	}
	if answer.Type != webrtc.SDPTypeAnswer || answer.SDP == "" {
		t.Fatalf("invalid companion answer: %+v", answer)
	}
}

func TestPionPeerDispatchesDataMessages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	received := make(chan string, 1)
	companion, err := NewPionPeerWithDataHandler(ctx, webrtc.Configuration{}, func(label string, payload []byte) {
		if label == "data" {
			received <- string(payload)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer companion.Close()

	browser, err := webrtc.NewAPI().NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer browser.Close()
	browser.OnDataChannel(func(channel *webrtc.DataChannel) {
		if channel.Label() == "data" {
			channel.OnOpen(func() { _ = channel.SendText("data-lane-message") })
		}
	})
	if _, err := browser.CreateDataChannel("browser-init", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := browser.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		t.Fatal(err)
	}
	offer, err := browser.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := browser.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	select {
	case <-webrtc.GatheringCompletePromise(browser):
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	local := browser.LocalDescription()
	if local == nil {
		t.Fatal("browser offer was not gathered")
	}
	answer, err := companion.Answer(ctx, *local)
	if err != nil {
		t.Fatal(err)
	}
	if err := browser.SetRemoteDescription(answer); err != nil {
		t.Fatal(err)
	}
	select {
	case payload := <-received:
		if payload != "data-lane-message" {
			t.Fatalf("payload=%q", payload)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
