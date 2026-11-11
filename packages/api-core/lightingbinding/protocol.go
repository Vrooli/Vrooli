// Package lightingbinding defines the private native Light Desk owner protocol.
// It carries no user credentials and is never mounted on the browser API.
package lightingbinding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/localprincipal"
	"github.com/vrooli/api-core/runtimediagnostics"
)

const Version = "lighting-binding-v1"

type Identity = runtimediagnostics.Identity

type Binding struct {
	Address     string `json:"address"`
	Incarnation string `json:"incarnation"`
}

func (b Binding) Validate() error {
	address, err := netip.ParseAddrPort(b.Address)
	if err != nil || !address.Addr().IsLoopback() || address.Addr().Zone() != "" || address.Port() == 0 || strings.TrimSpace(b.Incarnation) == "" || strings.ContainsRune(b.Incarnation, 0) || len(b.Incarnation) > 512 {
		return errors.New("exact loopback endpoint and backend incarnation are required")
	}
	return nil
}

type State struct {
	Version          string   `json:"version"`
	Identity         Identity `json:"identity"`
	Revision         uint64   `json:"revision"`
	Binding          *Binding `json:"binding,omitempty"`
	ReconcilePending bool     `json:"reconcile_pending"`
}

// Request is a compare-and-swap against the current ephemeral writer state.
// An identical retry of the most recent request returns its retained receipt.
// Earlier revisions can never reinstate an endpoint after quiesce/replacement.
type Request struct {
	Identity         Identity `json:"identity"`
	ExpectedRevision uint64   `json:"expected_revision"`
	OperationID      string   `json:"operation_id"`
	Binding          *Binding `json:"binding,omitempty"`
}

type Receipt struct {
	State       State        `json:"state"`
	OperationID string       `json:"operation_id"`
	Error       string       `json:"error,omitempty"`
	Zones       []ZoneResult `json:"zones,omitempty"`
}

type ZoneResult struct {
	ControllerID        string `json:"controller_id"`
	ZoneID              string `json:"zone_id"`
	Operation           int32  `json:"operation"`
	Reason              string `json:"reason"`
	LocalWriteCompleted bool   `json:"local_write_completed"`
	BackendAcknowledged bool   `json:"backend_acknowledged"`
	BackendAckStatus    uint32 `json:"backend_ack_status"`
	CachedModeReadback  int32  `json:"cached_mode_readback"`
}

func Directory(diagnosticsDirectory string) string {
	return filepath.Join(diagnosticsDirectory, "lighting")
}
func SocketPath(directory string, identity Identity) string {
	return runtimediagnostics.SocketPath(directory, identity)
}

var ErrConflict = errors.New("native lighting revision conflict")

type Client struct{ http *http.Client }

func NewClient(socket string, expectedPID int) *Client {
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
		if err != nil {
			return nil, err
		}
		unixConn, ok := conn.(*net.UnixConn)
		if !ok {
			_ = conn.Close()
			return nil, errors.New("native lighting requires a Unix peer")
		}
		peerPID, err := localprincipal.PeerPID(unixConn)
		if err != nil || expectedPID <= 0 || int(peerPID) != expectedPID {
			_ = conn.Close()
			return nil, errors.New("native lighting listener process identity mismatch or unavailable")
		}
		return conn, nil
	}}
	return &Client{http: &http.Client{Transport: transport, Timeout: 15 * time.Second}}
}
func (c *Client) Close() { c.http.CloseIdleConnections() }
func (c *Client) State(ctx context.Context) (State, error) {
	var state State
	err := c.call(ctx, http.MethodGet, "/v1/state", nil, &state)
	return state, err
}
func (c *Client) Apply(ctx context.Context, request Request) (Receipt, error) {
	var receipt Receipt
	err := c.call(ctx, http.MethodPost, "/v1/apply", request, &receipt)
	if err == nil && receipt.Error != "" {
		err = errors.New(receipt.Error)
	}
	return receipt, err
}
func (c *Client) call(ctx context.Context, method, path string, body any, output any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, "http://native-lighting"+path, reader)
	if err != nil {
		return err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusConflict {
		return ErrConflict
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("native lighting owner returned HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("invalid native lighting response envelope")
	}
	return nil
}
