package main

import (
	"context"
	"log"
	"os"
	"runtime"

	"vrooli-bridge/agent/internal/storeunlock"

	"github.com/vrooli/platform-go/credentialunlock"
)

// startStoreUnlockServer serves this node's credential-store passphrase to
// local Vrooli processes running as the agent's user. The control plane
// escrows the passphrase and delivers it sealed on every connection, so a
// headless node reopens its store after a reboot with nobody present; until a
// delivery arrives the server answers "not delivered yet". Windows opens its
// store through DPAPI and has no peer credentials, so it is skipped there.
func startStoreUnlockServer(ctx context.Context, logger *log.Logger, holder *storeunlock.Holder) {
	if runtime.GOOS == "windows" || holder == nil {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Printf("credential unlock: home directory unavailable, store hand-off disabled: %v", err)
		return
	}
	socket, err := credentialunlock.SocketPath(home)
	if err != nil {
		logger.Printf("credential unlock: socket path unavailable, store hand-off disabled: %v", err)
		return
	}
	listener, err := credentialunlock.Listen(socket)
	if err != nil {
		logger.Printf("credential unlock: listen %s failed, store hand-off disabled: %v", socket, err)
		return
	}
	logger.Printf("credential unlock: serving this node's store passphrase to its own user at %s", socket)
	go func() {
		if err := credentialunlock.Serve(ctx, listener, holder.Get); err != nil {
			logger.Printf("credential unlock: server stopped: %v", err)
		}
	}()
}
