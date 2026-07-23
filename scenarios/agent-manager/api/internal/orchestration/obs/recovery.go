package obs

import (
	"fmt"
	"runtime/debug"
)

// PanicFailure is the durable diagnostic payload supplied to an execution
// owner's failure transition after a goroutine panic. The stack is captured at
// the recovery boundary, before defer unwinding loses the useful frames.
type PanicFailure struct {
	Operation string
	Value     any
	Stack     string
}

// Error returns a stable error suitable for an existing terminal-failure path.
func (f PanicFailure) Error() string {
	return fmt.Sprintf("panic in %s: %v", f.Operation, f.Value)
}

// RecoverToFailure converts a recovered goroutine panic into an observable
// failure callback. Call it directly from a defer at every execution-goroutine
// boundary. The callback owns the domain-specific durable transition (run or
// workflow execution), while this package owns consistent stack capture and
// structured logging.
func RecoverToFailure(operation string, onFailure func(PanicFailure)) {
	value := recover()
	if value == nil {
		return
	}
	failure := PanicFailure{
		Operation: operation,
		Value:     value,
		Stack:     string(debug.Stack()),
	}
	Logger().Error("execution panic recovered", "operation", operation, "panic", fmt.Sprint(value), "stack", failure.Stack)
	if onFailure != nil {
		onFailure(failure)
	}
}
