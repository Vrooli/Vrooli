package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	adminhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/logx"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
)

// NOTE: TestHandleAdminResetDemoData_Success has been removed - depends on variants table which was removed
// Variant configuration is now stored in JSON files and managed via ConfigStore

func TestHandleAdminResetDemoData_ResetErrorReturns500(t *testing.T) {
	handler := adminhttp.NewResetConnectHandler(adminhttp.ResetDependencies{Reset: func(context.Context) error { return errors.New("database credentials") }, Now: time.Now, LogError: logx.Error})
	_, err := handler.ResetDemoData(context.Background(), connect.NewRequest(&lpbsv1.ResetDemoDataRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal || err == nil || strings.Contains(err.Error(), "credentials") {
		t.Fatalf("expected redacted internal error, got %v", err)
	}
}
