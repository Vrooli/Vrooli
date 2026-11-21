package signals

import "context"

// Repository intentionally exposes append and reads only. No update or delete
// operation exists because the journal's immutability is structural.
type Repository interface {
	Append(ctx context.Context, signal Signal) (CaptureResult, error)
	Get(ctx context.Context, id string) (Signal, error)
	List(ctx context.Context, limit int) ([]Signal, error)
}
