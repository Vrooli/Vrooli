package desktopwebrtc

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
)

// PionPeer is the companion-side WebRTC transport. Device Control still
// decides whether a caller may mutate the desktop; these channels carry only
// already-admitted protocol messages and selected-display VP8 samples.
type PionPeer struct {
	mu           sync.Mutex
	pc           *webrtc.PeerConnection
	video        *webrtc.TrackLocalStaticSample
	control      *webrtc.DataChannel
	data         *webrtc.DataChannel
	videoQueue   *BoundedQueue[media.Sample]
	controlQueue *BoundedQueue[[]byte]
	dataQueue    *BoundedQueue[[]byte]
	cancel       context.CancelFunc
	closed       bool
	onFailure    func()
	failureOnce  sync.Once
	statsMu      sync.Mutex
	stats        transportCounters
}

type transportCounters struct {
	framesSent, bytesSent, framesDropped                 uint64
	controlSent, dataSent, controlRejected, dataRejected uint64
}

// DataMessageHandler receives opaque, bounded messages after the WebRTC data
// channel has been negotiated. Device Control remains responsible for parsing
// and authorizing the message; Pion only owns channel delivery.
type DataMessageHandler func(label string, payload []byte)

func NewPionPeer(ctx context.Context) (*PionPeer, error) {
	return NewPionPeerWithDataHandler(ctx, webrtc.Configuration{}, nil)
}

// NewPionPeerWithConfiguration creates the companion-side peer with the
// deployment-selected ICE routes. The default constructor remains direct-ICE
// only; production callers may provide short-lived TURN servers without
// persisting credentials in the session manager.
func NewPionPeerWithConfiguration(ctx context.Context, configuration webrtc.Configuration) (*PionPeer, error) {
	return NewPionPeerWithDataHandler(ctx, configuration, nil)
}

func NewPionPeerWithDataHandler(ctx context.Context, configuration webrtc.Configuration, handler DataMessageHandler) (*PionPeer, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	mediaEngine := &webrtc.MediaEngine{}
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, PayloadType: 96}, webrtc.RTPCodecTypeVideo); err != nil {
		return nil, err
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine))
	pc, err := api.NewPeerConnection(configuration)
	if err != nil {
		return nil, err
	}
	track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, "desktop", "vrooli")
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	if _, err := pc.AddTrack(track); err != nil {
		_ = pc.Close()
		return nil, err
	}
	control, err := pc.CreateDataChannel("control", &webrtc.DataChannelInit{Ordered: boolPtr(true)})
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	data, err := pc.CreateDataChannel("data", &webrtc.DataChannelInit{Ordered: boolPtr(true)})
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	transportCtx, cancel := context.WithCancel(ctx)
	p := &PionPeer{
		pc:           pc,
		video:        track,
		control:      control,
		data:         data,
		videoQueue:   NewBoundedQueue[media.Sample](3),
		controlQueue: NewBoundedQueue[[]byte](32),
		dataQueue:    NewBoundedQueue[[]byte](64),
		cancel:       cancel,
	}
	if handler != nil {
		control.OnMessage(func(message webrtc.DataChannelMessage) { handler("control", append([]byte(nil), message.Data...)) })
		data.OnMessage(func(message webrtc.DataChannelMessage) { handler("data", append([]byte(nil), message.Data...)) })
	}
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			p.notifyFailure()
			_ = p.Close()
		}
	})
	go p.drainVideo(transportCtx)
	go p.drainData(transportCtx, p.control, p.controlQueue)
	go p.drainData(transportCtx, p.data, p.dataQueue)
	go func() { <-transportCtx.Done(); _ = p.Close() }()
	return p, nil
}

// SetFailureHandler installs the session-owner callback for failures that
// originate below the session manager, such as a browser disconnect or a
// failed native WebRTC write. The callback is invoked at most once.
func (p *PionPeer) SetFailureHandler(handler func()) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.onFailure = handler
	p.mu.Unlock()
}

func (p *PionPeer) notifyFailure() {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	handler := p.onFailure
	p.mu.Unlock()
	if handler != nil {
		p.failureOnce.Do(handler)
	}
}

func (p *PionPeer) Offer(ctx context.Context) (webrtc.SessionDescription, error) {
	if p == nil || p.pc == nil {
		return webrtc.SessionDescription{}, errors.New("peer is unavailable")
	}
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	if err := p.pc.SetLocalDescription(offer); err != nil {
		return webrtc.SessionDescription{}, err
	}
	select {
	case <-webrtc.GatheringCompletePromise(p.pc):
	case <-ctx.Done():
		return webrtc.SessionDescription{}, ctx.Err()
	}
	if local := p.pc.LocalDescription(); local != nil {
		return *local, nil
	}
	return webrtc.SessionDescription{}, errors.New("local offer unavailable")
}

// Answer accepts the browser offer and returns a fully gathered companion
// answer. The browser is the signaling initiator so Bridge can carry the
// bounded offer/answer/ICE payloads without Bridge owning the peer.
func (p *PionPeer) Answer(ctx context.Context, offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if p == nil || p.pc == nil {
		return webrtc.SessionDescription{}, errors.New("peer is unavailable")
	}
	if err := p.pc.SetRemoteDescription(offer); err != nil {
		return webrtc.SessionDescription{}, err
	}
	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	if err := p.pc.SetLocalDescription(answer); err != nil {
		return webrtc.SessionDescription{}, err
	}
	select {
	case <-webrtc.GatheringCompletePromise(p.pc):
	case <-ctx.Done():
		return webrtc.SessionDescription{}, ctx.Err()
	}
	if local := p.pc.LocalDescription(); local != nil {
		return *local, nil
	}
	return webrtc.SessionDescription{}, errors.New("local answer unavailable")
}

func (p *PionPeer) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	if p == nil || p.pc == nil {
		return errors.New("peer is unavailable")
	}
	return p.pc.AddICECandidate(candidate)
}

func (p *PionPeer) AddVP8Sample(ctx context.Context, payload []byte, duration time.Duration) error {
	if p == nil || p.video == nil || len(payload) == 0 {
		return errors.New("VP8 sample is unavailable")
	}
	if len(payload) > 32<<20 {
		return errors.New("VP8 sample exceeds limit")
	}
	if duration <= 0 {
		duration = 33 * time.Millisecond
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	dropped, err := p.videoQueue.Push(media.Sample{Data: append([]byte(nil), payload...), Duration: duration}, true)
	if dropped {
		p.statsMu.Lock()
		p.stats.framesDropped++
		p.statsMu.Unlock()
	}
	return err
}

func (p *PionPeer) WriteControl(payload []byte) error {
	return p.enqueueData(p.controlQueue, payload, true)
}

func (p *PionPeer) WriteData(payload []byte) error {
	return p.enqueueData(p.dataQueue, payload, false)
}

func (p *PionPeer) enqueueData(queue *BoundedQueue[[]byte], payload []byte, control bool) error {
	if p == nil || queue == nil || len(payload) == 0 || len(payload) > MaxDataMessageBytes {
		return errors.New("data payload is unavailable")
	}
	_, err := queue.Push(append([]byte(nil), payload...), false)
	if err != nil {
		p.statsMu.Lock()
		if control {
			p.stats.controlRejected++
		} else {
			p.stats.dataRejected++
		}
		p.statsMu.Unlock()
	}
	return err
}

func (p *PionPeer) drainVideo(ctx context.Context) {
	for {
		sample, ok := p.videoQueue.PopWait(ctx)
		if !ok {
			return
		}
		if err := p.video.WriteSample(sample); err != nil {
			p.notifyFailure()
			_ = p.Close()
			return
		}
		p.statsMu.Lock()
		p.stats.framesSent++
		p.stats.bytesSent += uint64(len(sample.Data))
		p.statsMu.Unlock()
	}
}

func (p *PionPeer) drainData(ctx context.Context, channel *webrtc.DataChannel, queue *BoundedQueue[[]byte]) {
	for {
		payload, ok := queue.PopWait(ctx)
		if !ok {
			return
		}
		for channel.ReadyState() != webrtc.DataChannelStateOpen {
			if channel.ReadyState() == webrtc.DataChannelStateClosed {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Millisecond):
			}
		}
		if err := channel.Send(payload); err != nil {
			p.notifyFailure()
			_ = p.Close()
			return
		}
		p.statsMu.Lock()
		if channel.Label() == "control" {
			p.stats.controlSent++
		} else {
			p.stats.dataSent++
		}
		p.statsMu.Unlock()
	}
}

// Stats returns bounded operational counters without exposing media, input,
// clipboard, SDP, or ICE payloads.
func (p *PionPeer) Stats() *desktopv1.DesktopTransportStats {
	if p == nil {
		return nil
	}
	p.statsMu.Lock()
	defer p.statsMu.Unlock()
	return &desktopv1.DesktopTransportStats{
		FramesSent:              p.stats.framesSent,
		BytesSent:               p.stats.bytesSent,
		FramesDropped:           p.stats.framesDropped,
		ControlMessagesSent:     p.stats.controlSent,
		DataMessagesSent:        p.stats.dataSent,
		ControlMessagesRejected: p.stats.controlRejected,
		DataMessagesRejected:    p.stats.dataRejected,
	}
}

func (p *PionPeer) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	cancel := p.cancel
	pc := p.pc
	videoQueue, controlQueue, dataQueue := p.videoQueue, p.controlQueue, p.dataQueue
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	videoQueue.Close()
	controlQueue.Close()
	dataQueue.Close()
	if pc != nil {
		return pc.Close()
	}
	return nil
}
func boolPtr(v bool) *bool { return &v }
