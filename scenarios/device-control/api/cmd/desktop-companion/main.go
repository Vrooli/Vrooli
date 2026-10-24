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
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"device-control/handlers/desktop"
	internalclock "device-control/internal/clock"
	"device-control/internal/desktopwebrtc"
	"device-control/internal/native/macos"
	"device-control/internal/server"
)

var companionVersion = "0.1.0"

func main() {
	configPath := flag.String("config", "", "user-scoped companion configuration path")
	flag.Parse()
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

	port := strings.TrimSpace(os.Getenv("DEVICE_CONTROL_COMPANION_PORT"))
	if port == "" {
		port = strings.TrimSpace(os.Getenv("API_PORT"))
	}
	if port == "" {
		port = "16465"
	}
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("invalid companion port %q", port)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	srv := server.New(server.Deps{Clock: internalclock.System(), Logger: log.Default()}, desktop.Module(desktopwebrtc.NewManager(provider)))
	httpServer := &http.Server{Addr: "127.0.0.1:" + port, Handler: srv.Handler()} // #nosec G112 -- loopback-only companion boundary.
	go func() {
		<-ctx.Done()
		_ = httpServer.Shutdown(context.Background())
	}()
	log.Printf("device-control companion listening on %s", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
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
