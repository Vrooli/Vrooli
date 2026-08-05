package procmetrics

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// xdotoolShell builds a mock shell that handles which, search (--pid/--name), and getwindowgeometry.
type xdotoolMock struct {
	pidOut     string
	pidErr     error
	displayOut string
	displayErr error
	// Map of windowID → geometry shell output
	geometries map[string]string
}

func (m *xdotoolMock) shell(_ context.Context, _ []string, name string, args ...string) ([]byte, error) {
	if name == "which" {
		return []byte("/usr/bin/xdotool"), nil
	}
	if name == "xdotool" && len(args) > 0 {
		switch args[0] {
		case "search":
			for _, a := range args {
				if a == "--pid" {
					if m.pidErr != nil {
						return nil, m.pidErr
					}
					return []byte(m.pidOut), nil
				}
				if a == "--name" {
					if m.displayErr != nil {
						return nil, m.displayErr
					}
					return []byte(m.displayOut), nil
				}
			}
		case "getwindowgeometry":
			if len(args) >= 3 {
				windowID := args[2]
				if geo, ok := m.geometries[windowID]; ok {
					return []byte(geo), nil
				}
				return []byte("WINDOW=" + windowID + "\nX=0\nY=0\nWIDTH=1280\nHEIGHT=720\n"), nil
			}
		case "getwindowname":
			return []byte("Application"), nil
		case "getwindowclassname":
			return []byte("Application"), nil
		}
	}
	return nil, fmt.Errorf("unexpected call")
}

func TestXdotoolDetector_HasVisibleWindow(t *testing.T) {
	tests := []struct {
		name       string
		pidOut     string
		pidErr     error
		displayOut string
		displayErr error
		wantResult bool
	}{
		{
			name:       "window found via PID search",
			pidOut:     "12345678\n",
			wantResult: true,
		},
		{
			name:       "multiple windows via PID search",
			pidOut:     "12345678\n87654321\n",
			wantResult: true,
		},
		{
			name:       "PID search fails but display fallback finds window",
			pidErr:     fmt.Errorf("exit status 1"),
			displayOut: "99999\n",
			wantResult: true,
		},
		{
			name:       "PID search empty but display fallback finds window",
			pidOut:     "",
			displayOut: "99999\n",
			wantResult: true,
		},
		{
			name:       "no windows anywhere",
			pidErr:     fmt.Errorf("exit status 1"),
			displayErr: fmt.Errorf("exit status 1"),
			wantResult: false,
		},
		{
			name:       "both searches return empty",
			pidOut:     "",
			displayOut: "",
			wantResult: false,
		},
		{
			name:       "PID search whitespace only, display fallback finds window",
			pidOut:     "  \n  ",
			displayOut: "12345\n",
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &xdotoolMock{
				pidOut:     tt.pidOut,
				pidErr:     tt.pidErr,
				displayOut: tt.displayOut,
				displayErr: tt.displayErr,
			}
			d := NewXdotoolDetector(mock.shell, testLogger())
			result, err := d.HasVisibleWindow(context.Background(), 1234, ":99")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.wantResult {
				t.Errorf("got %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestXdotoolDetector_RejectsTinyDesktopHelperAsVisibleWindow(t *testing.T) {
	mock := &xdotoolMock{
		pidOut: "111\n",
		geometries: map[string]string{
			"111": "WINDOW=111\nX=-100\nY=-100\nWIDTH=1\nHEIGHT=1\n",
		},
	}
	d := NewXdotoolDetector(mock.shell, testLogger())

	visible, err := d.HasVisibleWindow(context.Background(), 1234, ":99")
	if err != nil {
		t.Fatalf("HasVisibleWindow returned error: %v", err)
	}
	if visible {
		t.Fatal("tiny desktop helper must not count as a visible application window")
	}

	geometry, err := d.LargestVisibleWindow(context.Background(), 1234, ":99")
	if err != nil {
		t.Fatalf("LargestVisibleWindow returned error: %v", err)
	}
	if geometry != nil {
		t.Fatalf("tiny desktop helper returned as usable geometry: %+v", geometry)
	}
}

func TestXdotoolDetector_RejectsFullScreenDesktopBackgroundAsApplication(t *testing.T) {
	shell := func(_ context.Context, _ []string, name string, args ...string) ([]byte, error) {
		if name == "which" {
			return []byte("/usr/bin/xdotool"), nil
		}
		if name != "xdotool" || len(args) == 0 {
			return nil, fmt.Errorf("unexpected call: %s %v", name, args)
		}
		switch args[0] {
		case "search":
			return []byte("111\n"), nil
		case "getwindowgeometry":
			return []byte("WINDOW=111\nX=0\nY=0\nWIDTH=1920\nHEIGHT=1080\n"), nil
		case "getwindowname":
			return []byte("Desktop"), nil
		case "getwindowclassname":
			return []byte("Desktop"), nil
		default:
			return nil, fmt.Errorf("unexpected xdotool command: %v", args)
		}
	}
	d := NewXdotoolDetector(shell, testLogger())

	visible, err := d.HasVisibleWindow(context.Background(), 1234, ":99")
	if err != nil {
		t.Fatalf("HasVisibleWindow returned error: %v", err)
	}
	if visible {
		t.Fatal("desktop background must not count as a visible application window")
	}
	geometry, err := d.LargestVisibleWindow(context.Background(), 1234, ":99")
	if err != nil {
		t.Fatalf("LargestVisibleWindow returned error: %v", err)
	}
	if geometry != nil {
		t.Fatalf("desktop background returned as application geometry: %+v", geometry)
	}
}

func TestXdotoolDetector_AcceptsWindowWhenClassLookupIsUnsupported(t *testing.T) {
	shell := func(_ context.Context, _ []string, name string, args ...string) ([]byte, error) {
		if name == "which" {
			return []byte("/usr/bin/xdotool"), nil
		}
		if name != "xdotool" || len(args) == 0 {
			return nil, fmt.Errorf("unexpected call: %s %v", name, args)
		}
		switch args[0] {
		case "search":
			return []byte("111\n"), nil
		case "getwindowgeometry":
			return []byte("WINDOW=111\nX=0\nY=0\nWIDTH=1200\nHEIGHT=800\n"), nil
		case "getwindowname":
			return []byte("Hello Desktop"), nil
		case "getwindowclassname":
			return nil, fmt.Errorf("xdotool: Unknown command: getwindowclassname")
		default:
			return nil, fmt.Errorf("unexpected xdotool command: %v", args)
		}
	}

	d := NewXdotoolDetector(shell, testLogger())
	visible, err := d.HasVisibleWindow(context.Background(), 1234, ":99")
	if err != nil {
		t.Fatalf("HasVisibleWindow returned error: %v", err)
	}
	if !visible {
		t.Fatal("supported window title should be sufficient when class lookup is unavailable")
	}
}

func TestXdotoolDetector_LargestVisibleWindow(t *testing.T) {
	tests := []struct {
		name       string
		pidOut     string
		geometries map[string]string
		wantW      int
		wantH      int
		wantNil    bool
	}{
		{
			name:   "single window",
			pidOut: "111\n",
			geometries: map[string]string{
				"111": "WINDOW=111\nX=0\nY=0\nWIDTH=1280\nHEIGHT=720\n",
			},
			wantW: 1280,
			wantH: 720,
		},
		{
			name:   "returns largest of multiple windows",
			pidOut: "111\n222\n",
			geometries: map[string]string{
				"111": "WINDOW=111\nX=0\nY=0\nWIDTH=300\nHEIGHT=200\n",
				"222": "WINDOW=222\nX=0\nY=0\nWIDTH=1920\nHEIGHT=1080\n",
			},
			wantW: 1920,
			wantH: 1080,
		},
		{
			name:    "no windows",
			pidOut:  "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &xdotoolMock{
				pidOut:     tt.pidOut,
				geometries: tt.geometries,
			}
			d := NewXdotoolDetector(mock.shell, testLogger())
			geo, err := d.LargestVisibleWindow(context.Background(), 1234, ":99")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if geo != nil {
					t.Errorf("expected nil, got %+v", geo)
				}
				return
			}
			if geo == nil {
				t.Fatal("expected non-nil geometry")
			}
			if geo.Width != tt.wantW || geo.Height != tt.wantH {
				t.Errorf("got %dx%d, want %dx%d", geo.Width, geo.Height, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestXdotoolDetector_NotInstalled(t *testing.T) {
	shell := func(_ context.Context, _ []string, name string, _ ...string) ([]byte, error) {
		if name == "which" {
			return nil, fmt.Errorf("exit status 1")
		}
		t.Fatal("xdotool should not be called when not installed")
		return nil, nil
	}

	d := NewXdotoolDetector(shell, testLogger())
	result, err := d.HasVisibleWindow(context.Background(), 1234, ":99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result {
		t.Error("expected false when xdotool not installed")
	}

	geo, err := d.LargestVisibleWindow(context.Background(), 1234, ":99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if geo != nil {
		t.Error("expected nil geometry when xdotool not installed")
	}
}

func TestXdotoolDetector_CachesAvailability(t *testing.T) {
	whichCalls := 0
	shell := func(_ context.Context, _ []string, name string, _ ...string) ([]byte, error) {
		if name == "which" {
			whichCalls++
			return []byte("/usr/bin/xdotool"), nil
		}
		return []byte(""), nil
	}

	d := NewXdotoolDetector(shell, testLogger())
	ctx := context.Background()

	_, _ = d.HasVisibleWindow(ctx, 1, ":99")
	_, _ = d.HasVisibleWindow(ctx, 2, ":99")
	_, _ = d.HasVisibleWindow(ctx, 3, ":99")

	if whichCalls != 1 {
		t.Errorf("which called %d times, want 1 (should be cached)", whichCalls)
	}
}

func TestParseGeometryShell(t *testing.T) {
	tests := []struct {
		name  string
		input string
		wantW int
		wantH int
	}{
		{
			name:  "normal output",
			input: "WINDOW=12345678\nX=0\nY=0\nWIDTH=1280\nHEIGHT=720\n",
			wantW: 1280,
			wantH: 720,
		},
		{
			name:  "large dimensions",
			input: "WINDOW=99\nX=100\nY=50\nWIDTH=1920\nHEIGHT=1080\n",
			wantW: 1920,
			wantH: 1080,
		},
		{
			name:  "empty input",
			input: "",
			wantW: 0,
			wantH: 0,
		},
		{
			name:  "missing HEIGHT",
			input: "WINDOW=1\nWIDTH=800\n",
			wantW: 800,
			wantH: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := parseGeometryShell(tt.input)
			if w != tt.wantW || h != tt.wantH {
				t.Errorf("got %dx%d, want %dx%d", w, h, tt.wantW, tt.wantH)
			}
		})
	}
}
