package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/projectmeta"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	"github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	"google.golang.org/protobuf/proto"
)

type postgresTestPoolRoutingService struct {
	delegate routing_v1connect.RoutingServiceHandler
	primary  string
	mu       sync.Mutex
	leases   map[string]string
}

func registerScenarioDevRouting(router *mux.Router, routedDB *database.RoutedDB, roots *filerouting.RoutedRoots) error {
	if !projectmeta.IsDevelopment() && os.Getenv(apihttp.TestModeForceEnableEnv) != "1" {
		return nil
	}
	primary, err := resolveDatabaseURL()
	if err != nil {
		return err
	}
	service := &postgresTestPoolRoutingService{delegate: devrouting.NewService(routedDB, roots), primary: primary, leases: make(map[string]string)}
	if !devrouting.RegisterWithFileRootsService(devRoutingMux{router: router}, routedDB, roots, service) {
		return fmt.Errorf("development routing is disabled or incomplete")
	}
	return nil
}

func (s *postgresTestPoolRoutingService) InstallTestPool(ctx context.Context, req *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	leaseID := strings.TrimSpace(req.Msg.GetLeaseId())
	if leaseID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("lease_id is required"))
	}
	poolDSN, databaseName := req.Msg.GetDsn(), ""
	if isSQLitePlaceholder(poolDSN) {
		var err error
		databaseName, poolDSN, err = s.createLeaseDatabase(ctx, leaseID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create postgres test pool: %w", err))
		}
	}
	message := proto.Clone(req.Msg).(*routingv1.InstallTestPoolRequest)
	message.Dsn = poolDSN
	response, err := s.delegate.InstallTestPool(ctx, connect.NewRequest(message))
	if err != nil {
		if databaseName != "" {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			_ = s.dropLeaseDatabase(cleanupCtx, databaseName)
		}
		return nil, err
	}
	if databaseName != "" {
		s.mu.Lock()
		s.leases[leaseID] = databaseName
		s.mu.Unlock()
	}
	return response, nil
}

func (s *postgresTestPoolRoutingService) ClearTestPool(ctx context.Context, req *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	response, err := s.delegate.ClearTestPool(ctx, req)
	if err != nil {
		return nil, err
	}
	leaseID := strings.TrimSpace(req.Msg.GetLeaseId())
	s.mu.Lock()
	databaseName := s.leases[leaseID]
	delete(s.leases, leaseID)
	s.mu.Unlock()
	if databaseName != "" {
		if err := s.dropLeaseDatabase(ctx, databaseName); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("drop postgres test pool: %w", err))
		}
	}
	return response, nil
}

func (s *postgresTestPoolRoutingService) HeartbeatTestPool(ctx context.Context, req *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error) {
	return s.delegate.HeartbeatTestPool(ctx, req)
}

func isSQLitePlaceholder(dsn string) bool {
	return strings.HasPrefix(strings.TrimSpace(strings.ToLower(dsn)), "file:")
}

func leaseDatabaseName(leaseID string) (string, error) {
	var builder strings.Builder
	builder.WriteString("lpbs_workflow_")
	for _, character := range strings.ToLower(leaseID) {
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			builder.WriteRune(character)
		default:
			builder.WriteByte('_')
		}
	}
	name := strings.Trim(builder.String(), "_")
	if len(name) == 0 || len(name) > 63 {
		return "", fmt.Errorf("invalid lease id")
	}
	return name, nil
}

func (s *postgresTestPoolRoutingService) createLeaseDatabase(ctx context.Context, leaseID string) (string, string, error) {
	name, err := leaseDatabaseName(leaseID)
	if err != nil {
		return "", "", err
	}
	adminURL, poolURL, err := postgresLeaseURLs(s.primary, name)
	if err != nil {
		return "", "", err
	}
	admin, err := sql.Open("postgres", adminURL)
	if err != nil {
		return "", "", err
	}
	defer admin.Close()
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
		return "", "", err
	}
	return name, poolURL, nil
}

func (s *postgresTestPoolRoutingService) dropLeaseDatabase(ctx context.Context, name string) error {
	adminURL, _, err := postgresLeaseURLs(s.primary, name)
	if err != nil {
		return err
	}
	admin, err := sql.Open("postgres", adminURL)
	if err != nil {
		return err
	}
	defer admin.Close()
	_, err = admin.ExecContext(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
	return err
}

func postgresLeaseURLs(primary, databaseName string) (string, string, error) {
	parsed, err := url.Parse(primary)
	if err != nil {
		return "", "", fmt.Errorf("parse primary database URL: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", "", fmt.Errorf("primary database URL must use postgres")
	}
	pool, admin := *parsed, *parsed
	pool.Path, admin.Path = "/"+databaseName, "/postgres"
	return admin.String(), pool.String(), nil
}
