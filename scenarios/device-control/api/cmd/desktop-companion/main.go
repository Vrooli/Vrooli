// Command desktop-companion is the Device Control-owned user-session process
// installed by Bridge. It exposes only the typed desktop-session service on
// loopback; Bridge and the node agent remain the remote trust boundary.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"device-control/handlers/desktop"
	internalclock "device-control/internal/clock"
	"device-control/internal/desktopcompanion"
	"device-control/internal/desktopwebrtc"
	"device-control/internal/environment"
	"device-control/internal/native/macos"
	"device-control/internal/server"
	"github.com/vrooli/platform-go/desktopowner"
)

var companionVersion = "0.1.0"

func main() {
	processEnv := environment.OSReader{}
	configPath := flag.String("config", "", "user-scoped companion configuration path")
	sessionID := flag.String("x11-session", "", "exact local logind session ID or active-user")
	display := flag.Uint("x11-display", 0, "local X11 display number")
	authority := flag.String("x11-authority", "", "private Xauthority file path")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *configPath != "" {
		if err := validateConfig(*configPath); err != nil {
			log.Fatalf("desktop companion configuration failed: %v", err)
		}
	}

	provider := desktopwebrtc.Provider(desktopwebrtc.UnavailableProvider{})
	if runtime.GOOS == "darwin" {
		backend, err := macos.NewHostBackend("")
		if err != nil {
			log.Fatalf("desktop companion native backend failed: %v", err)
		}
		provider = macos.Provider{Backend: backend, CompanionVersion: companionVersion}
	}

	var listener net.Listener
	if runtime.GOOS == "linux" {
		if *sessionID == "" || *display > 65535 || !filepath.IsAbs(*authority) {
			log.Fatal("Linux companion requires an exact X11 session and private authority file")
		}
		var err error
		listener, err = activationListener()
		if err != nil {
			log.Fatal(err)
		}
		defer listener.Close()
		linux, err := desktopcompanion.NewX11(desktopcompanion.X11Config{
			Display: uint16(*display), AuthorityFile: *authority, SessionID: *sessionID,
		}, companionVersion)
		if err != nil {
			log.Fatalf("desktop X11 binding failed: %v", err)
		}
		defer linux.Shutdown()
		provider = linux
	}
	port := strings.TrimSpace(environment.Value(processEnv, "DEVICE_CONTROL_COMPANION_PORT"))
	if port == "" {
		port = strings.TrimSpace(environment.Value(processEnv, "API_PORT"))
	}
	if port == "" {
		port = "16465"
	}
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("invalid companion port %q", port)
	}

	manager := desktopwebrtc.NewManager(provider)
	go manager.Maintain(ctx)
	srv := server.New(server.Deps{Clock: internalclock.System(), Logger: log.Default()}, desktop.Module(manager))
	httpServer := &http.Server{
		Addr: "127.0.0.1:" + port, Handler: srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 45 * time.Second, WriteTimeout: 20 * time.Second,
	}
	go func() {
		<-ctx.Done()
		drain, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(drain); err != nil {
			_ = httpServer.Close()
		}
	}()
	var err error
	if listener != nil {
		log.Printf("device-control companion listening on owner socket")
		err = httpServer.Serve(listener)
	} else {
		err = httpServer.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("desktop companion listener failed: %v", err)
	}
}

func activationListener() (net.Listener, error) {
	if os.Getenv("LISTEN_PID") != strconv.Itoa(os.Getpid()) || os.Getenv("LISTEN_FDS") != "1" {
		return nil, errors.New("Linux companion requires one systemd-activated Unix listener")
	}
	file := os.NewFile(3, "desktop-owner-socket")
	defer file.Close()
	listener, err := net.FileListener(file)
	if err != nil {
		return nil, err
	}
	address, ok := listener.Addr().(*net.UnixAddr)
	if !ok || address.Name != desktopowner.SocketPath(os.Getuid()) {
		listener.Close()
		return nil, errors.New("invalid desktop owner socket")
	}
	info, err := os.Lstat(address.Name)
	if err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm() != 0600 {
		listener.Close()
		return nil, errors.New("desktop owner socket must have mode 0600")
	}
	return listener, nil
}

func validateConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %q: %w", path, err)
	}
	var config struct {
		DisplayID string `json:"display_id"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("decode %q: %w", path, err)
	}
	if len(config.DisplayID) > 256 {
		return fmt.Errorf("display_id exceeds 256 bytes")
	}
	return nil
}
