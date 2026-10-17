// This file carries existing inbound proof only across one CreateRun request.
package heartbeat

import (
	"context"
	"fmt"
	"strings"
)

type createRunCallerKey struct{}
type createRunCallerProof struct{ authorization, runIdentity string }

func withCreateRunCaller(ctx context.Context, authorization, runIdentity string) context.Context {
	return context.WithValue(ctx, createRunCallerKey{}, createRunCallerProof{authorization: authorization, runIdentity: runIdentity})
}
func createRunCaller(ctx context.Context) createRunCallerProof {
	proof, _ := ctx.Value(createRunCallerKey{}).(createRunCallerProof)
	return proof
}
func redactCreateRunCallerError(err error, proof createRunCallerProof) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	for _, secret := range []string{proof.authorization, proof.runIdentity, strings.TrimSpace(strings.TrimPrefix(proof.authorization, "Bearer "))} {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[redacted]")
		}
	}
	return fmt.Errorf("%s", message)
}
