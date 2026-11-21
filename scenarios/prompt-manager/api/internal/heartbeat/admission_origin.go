package heartbeat

import "context"

type heartbeatAdmissionOriginKey struct{}

func withHeartbeatAdmissionOrigin(ctx context.Context, origin string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, heartbeatAdmissionOriginKey{}, origin)
}

func heartbeatAdmissionOrigin(ctx context.Context) string {
	if ctx == nil {
		return AdmissionOriginUnknown
	}
	origin, _ := ctx.Value(heartbeatAdmissionOriginKey{}).(string)
	if origin == AdmissionOriginScheduled || origin == AdmissionOriginManual {
		return origin
	}
	return AdmissionOriginUnknown
}
