package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/vrooli/secrets-manager-native-host/protocol"
)

// The default host is intentionally fail-closed. A release must inject an
// authority adapter that verifies enrollment and resolves a grant-bound fill;
// it must never treat an absent adapter as a local credential store.
func main() {
	log.SetOutput(os.Stderr)
	host := protocol.NewHost(newHTTPAuthorityFromEnv())
	for {
		frame, err := protocol.ReadFrame(os.Stdin)
		if err != nil {
			if err != io.EOF {
				log.Printf("native host input failed: %v", err)
			}
			return
		}
		request, err := protocol.DecodeRequest(frame)
		if err != nil {
			writeFailure(err)
			continue
		}
		response := host.Handle(context.Background(), request)
		payload, err := protocol.EncodeResponse(response)
		if err != nil {
			writeFailure(err)
			continue
		}
		if err := protocol.WriteFrame(os.Stdout, payload); err != nil {
			log.Printf("native host output failed: %v", err)
			return
		}
	}
}

func writeFailure(err error) {
	response, marshalErr := json.Marshal(map[string]string{"protocol": protocol.ProtocolName, "ok": "false", "code": "invalid_request", "message": err.Error()})
	if marshalErr != nil {
		fmt.Fprintln(os.Stderr, "native host response failed:", marshalErr)
		return
	}
	if writeErr := protocol.WriteFrame(os.Stdout, response); writeErr != nil {
		fmt.Fprintln(os.Stderr, "native host response failed:", writeErr)
	}
}
