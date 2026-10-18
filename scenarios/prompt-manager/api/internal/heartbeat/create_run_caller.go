// This file carries existing inbound proof only across one CreateRun request.
package heartbeat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/owneridentity"
)

type createRunCallerKey struct{}
type createRunCallerProof struct {
	authorization, runIdentity string
	origin                     string
	binding                    string
}

const (
	createRunCallerOriginUnqualified = "unqualified"
	createRunCallerOriginCaller      = "caller"
	createRunCallerOriginService     = "service"
)

func withCreateRunCaller(ctx context.Context, authorization, runIdentity string) context.Context {
	origin := createRunCallerOriginUnqualified
	if authorization != "" || runIdentity != "" {
		origin = createRunCallerOriginCaller
	}
	qualified := withCreateRunCallerOrigin(ctx, authorization, runIdentity, origin)
	if origin == createRunCallerOriginCaller {
		if binding, err := owneridentity.CreateRunCallerBinding(qualified); err == nil {
			return withCreateRunCallerBinding(qualified, binding)
		}
	}
	return qualified
}

func withCreateRunCallerOrigin(ctx context.Context, authorization, runIdentity, origin string) context.Context {
	return context.WithValue(ctx, createRunCallerKey{}, createRunCallerProof{authorization: authorization, runIdentity: runIdentity, origin: origin})
}
func withCreateRunCallerBinding(ctx context.Context, binding string) context.Context {
	proof := createRunCaller(ctx)
	proof.binding = binding
	return context.WithValue(ctx, createRunCallerKey{}, proof)
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
	var response *AgentManagerHTTPError
	if errors.As(err, &response) && response != nil {
		prefix := "agent-manager error: "
		detail := strings.TrimPrefix(message, prefix)
		return &AgentManagerHTTPError{StatusCode: response.StatusCode, Code: response.Code, Message: detail, AdmissionReason: response.AdmissionReason}
	}
	return fmt.Errorf("%s", message)
}
