package livedesktop

import (
	"sync"
	"time"

	"scenario-to-desktop-api/procmetrics"
)

// SessionState represents the lifecycle state of a live desktop session.
type SessionState string

const (
	StateCreating SessionState = "creating"
	StateRunning  SessionState = "running"
	StateStopping SessionState = "stopping"
	StateStopped  SessionState = "stopped"
	StateError    SessionState = "error"
)

// Session holds the full state of a live desktop session.
type Session struct {
	ID            string             `json:"id"`
	ScenarioName  string             `json:"scenario_name"`
	State         SessionState       `json:"state"`
	Display       PlatformDisplay    `json:"-"`
	RemoteAccess  RemoteAccessHandle `json:"-"`
	RemoteInfo    RemoteAccessInfo   `json:"-"`
	AppProcess    PlatformProcess    `json:"-"`
	VNCPort       int                `json:"vnc_port"`
	WSPort        int                `json:"ws_port"`
	CreatedAt     time.Time          `json:"created_at"`
	LastHeartbeat time.Time          `json:"last_heartbeat"`
	Error         string             `json:"error,omitempty"`
	Width         int                `json:"width"`
	Height        int                `json:"height"`

	Platform string `json:"platform"`

	// Control state
	IsRecording   bool                `json:"-"`
	CaptureID     string              `json:"-"`
	NetworkMode   string              `json:"network_mode"`
	BandwidthKbps int                 `json:"bandwidth_kbps,omitempty"`
	EnvVars       map[string]string   `json:"-"`
	DarkMode      bool                `json:"dark_mode"`
	Locale        string              `json:"locale,omitempty"`
	AppRunning    bool                `json:"app_running"`
	Monitor       procmetrics.Monitor `json:"-"`

	mu       sync.Mutex
	stopOnce sync.Once
	stopCh   chan struct{}
}

// SetState updates the session state under lock.
func (s *Session) SetState(state SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = state
}

// SetError sets the session into error state with a message.
func (s *Session) SetError(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = StateError
	s.Error = msg
}

// Touch updates the last heartbeat timestamp.
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastHeartbeat = time.Now()
}

// Done is closed when the session is stopped. Background work that belongs to
// a session must use it instead of inheriting the request that created it.
func (s *Session) Done() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopCh
}

func (s *Session) signalStop() {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.stopCh == nil {
			s.stopCh = make(chan struct{})
		}
		close(s.stopCh)
	})
}

// SessionConfig is the configuration for creating a new session.
type SessionConfig struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	ScenarioName string `json:"scenario_name"`
	AppPath      string `json:"app_path,omitempty"`
	Platform     string `json:"platform,omitempty"`
}

// DOC: docs/reference/live-desktop-api.md#process-metrics
// MetricsView is the lightweight metrics snapshot included in session API responses.
type MetricsView struct {
	// SplashDurationMs is launch → first visible window (splash screen or immediate main window).
	SplashDurationMs *int64 `json:"splash_duration_ms,omitempty"`
	SplashDetected   bool   `json:"splash_detected"`

	// ReadyDurationMs is launch → main application window (meets size threshold).
	ReadyDurationMs *int64 `json:"ready_duration_ms,omitempty"`
	ReadyDetected   bool   `json:"ready_detected"`

	CurrentCPU   *float64 `json:"current_cpu_percent,omitempty"`
	CurrentRSSMB *float64 `json:"current_rss_mb,omitempty"`
	PeakRSSMB    *float64 `json:"peak_rss_mb,omitempty"`
	SampleCount  int      `json:"sample_count"`
}

// SessionView is the JSON-safe view of a session for API responses.
type SessionView struct {
	ID            string       `json:"id"`
	ScenarioName  string       `json:"scenario_name"`
	State         SessionState `json:"state"`
	VNCPort       int          `json:"vnc_port"`
	WSPort        int          `json:"ws_port"`
	Width         int          `json:"width"`
	Height        int          `json:"height"`
	CreatedAt     time.Time    `json:"created_at"`
	LastHeartbeat time.Time    `json:"last_heartbeat"`
	Error         string       `json:"error,omitempty"`
	IsRecording   bool         `json:"is_recording"`
	NetworkMode   string       `json:"network_mode"`
	BandwidthKbps int          `json:"bandwidth_kbps,omitempty"`
	DarkMode      bool         `json:"dark_mode"`
	Locale        string       `json:"locale,omitempty"`
	AppRunning    bool         `json:"app_running"`
	Platform      string       `json:"platform"`
	Metrics       *MetricsView `json:"metrics,omitempty"`
}

// View returns a JSON-safe snapshot of the session.
func (s *Session) View() SessionView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := SessionView{
		ID:            s.ID,
		ScenarioName:  s.ScenarioName,
		State:         s.State,
		VNCPort:       s.VNCPort,
		WSPort:        s.WSPort,
		Width:         s.Width,
		Height:        s.Height,
		CreatedAt:     s.CreatedAt,
		LastHeartbeat: s.LastHeartbeat,
		Error:         s.Error,
		IsRecording:   s.IsRecording,
		NetworkMode:   s.NetworkMode,
		BandwidthKbps: s.BandwidthKbps,
		DarkMode:      s.DarkMode,
		Locale:        s.Locale,
		AppRunning:    s.AppRunning,
		Platform:      s.Platform,
	}
	if s.Monitor != nil {
		v.Metrics = buildMetricsView(s.Monitor.Report())
	}
	return v
}

// buildMetricsView converts a procmetrics.Report to a lightweight MetricsView.
func buildMetricsView(r *procmetrics.Report) *MetricsView {
	if r == nil {
		return nil
	}
	mv := &MetricsView{
		SplashDurationMs: r.Startup.SplashDurationMs,
		SplashDetected:   r.Startup.SplashVisibleAt != nil,
		ReadyDurationMs:  r.Startup.ReadyMs,
		ReadyDetected:    r.Startup.ReadyAt != nil,
		SampleCount:      len(r.Samples),
	}
	if n := len(r.Samples); n > 0 {
		latest := r.Samples[n-1]
		cpu := latest.CPUPercent
		mv.CurrentCPU = &cpu
		rssMB := float64(latest.RSSBytes) / (1024 * 1024)
		mv.CurrentRSSMB = &rssMB
		peakMB := float64(latest.PeakBytes) / (1024 * 1024)
		mv.PeakRSSMB = &peakMB
	}
	return mv
}

// SetNetworkMode updates the network mode under lock.
func (s *Session) SetNetworkMode(mode string, bandwidthKbps int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.NetworkMode = mode
	s.BandwidthKbps = bandwidthKbps
}

// SetRecording updates the recording state under lock.
func (s *Session) SetRecording(recording bool, captureID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsRecording = recording
	s.CaptureID = captureID
}

// SetDarkMode updates the dark mode state under lock.
func (s *Session) SetDarkMode(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DarkMode = enabled
}

// SetLocale updates the locale under lock.
func (s *Session) SetLocale(locale string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Locale = locale
}

// SetAppRunning updates the app running state under lock.
func (s *Session) SetAppRunning(running bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AppRunning = running
}

// SetEnvVars sets the environment variables under lock.
func (s *Session) SetEnvVars(vars map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EnvVars = vars
}

// GetEnvVars returns a copy of the environment variables under lock.
func (s *Session) GetEnvVars() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.EnvVars == nil {
		return nil
	}
	cp := make(map[string]string, len(s.EnvVars))
	for k, v := range s.EnvVars {
		cp[k] = v
	}
	return cp
}

// SetMonitor sets the process monitor under lock.
func (s *Session) SetMonitor(m procmetrics.Monitor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Monitor = m
}

// GetMonitor returns the process monitor under lock.
func (s *Session) GetMonitor() procmetrics.Monitor {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Monitor
}
