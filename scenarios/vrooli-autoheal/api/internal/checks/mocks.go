// Package checks provides mock implementations for testing
// [REQ:TEST-SEAM-001] Mock implementations for testing
package checks

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"
)

// MockExecutor is a test implementation of CommandExecutor.
// It returns preconfigured responses based on the command.
type MockExecutor struct {
	// Responses maps "command arg1 arg2..." to the response
	Responses map[string]MockResponse
	// DefaultResponse is returned when no matching response is found
	DefaultResponse MockResponse
	// Calls records all command invocations for verification
	Calls []MockCall
}

// MockResponse represents a command response.
type MockResponse struct {
	Output []byte
	Error  error
}

// MockCall represents a recorded command call.
type MockCall struct {
	Name string
	Args []string
}

// commandKey generates a key for the Responses map.
func commandKey(name string, args []string) string {
	key := name
	for _, arg := range args {
		key += " " + arg
	}
	return key
}

// Output returns the mock response for the command.
func (m *MockExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, MockCall{Name: name, Args: args})
	key := commandKey(name, args)
	if resp, ok := m.Responses[key]; ok {
		return resp.Output, resp.Error
	}
	return m.DefaultResponse.Output, m.DefaultResponse.Error
}

// CombinedOutput returns the mock response for the command.
func (m *MockExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.Output(ctx, name, args...)
}

// Run returns the mock error for the command.
func (m *MockExecutor) Run(ctx context.Context, name string, args ...string) error {
	_, err := m.Output(ctx, name, args...)
	return err
}

// NewMockExecutor creates a new MockExecutor with empty responses.
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Responses: make(map[string]MockResponse),
	}
}

// MockHTTPClient is a test implementation of HTTPDoer.
type MockHTTPClient struct {
	// Responses maps URL to the response
	Responses map[string]MockHTTPResponse
	// DefaultResponse is returned when no matching response is found
	DefaultResponse MockHTTPResponse
	// Calls records all request URLs for verification
	Calls []string
}

// MockHTTPResponse represents an HTTP response.
type MockHTTPResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
	Error      error
}

// Do returns the mock response for the request.
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	m.Calls = append(m.Calls, url)

	var resp MockHTTPResponse
	if r, ok := m.Responses[url]; ok {
		resp = r
	} else {
		resp = m.DefaultResponse
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	httpResp := &http.Response{
		StatusCode: resp.StatusCode,
		Body:       io.NopCloser(bytes.NewBufferString(resp.Body)),
		Header:     make(http.Header),
	}
	for k, v := range resp.Headers {
		httpResp.Header.Set(k, v)
	}
	return httpResp, nil
}

// NewMockHTTPClient creates a new MockHTTPClient with empty responses.
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		Responses: make(map[string]MockHTTPResponse),
		DefaultResponse: MockHTTPResponse{
			StatusCode: 200,
			Body:       `{"status":"ok"}`,
		},
	}
}

// MockDialer is a test implementation of NetworkDialer.
type MockDialer struct {
	// Responses maps "address" to the response
	Responses map[string]MockDialResponse
	// DefaultResponse is returned when no matching response is found
	DefaultResponse MockDialResponse
	// Calls records all dial invocations for verification
	Calls []string
}

// MockDialResponse represents a dial response.
type MockDialResponse struct {
	Conn  net.Conn
	Error error
}

// DialTimeout returns the mock connection or error.
func (m *MockDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	m.Calls = append(m.Calls, address)
	if resp, ok := m.Responses[address]; ok {
		return resp.Conn, resp.Error
	}
	return m.DefaultResponse.Conn, m.DefaultResponse.Error
}

// NewMockDialer creates a new MockDialer with empty responses.
func NewMockDialer() *MockDialer {
	return &MockDialer{
		Responses: make(map[string]MockDialResponse),
	}
}

// MockConn is a minimal net.Conn implementation for testing.
type MockConn struct {
	closed bool
}

func (c *MockConn) Read(b []byte) (n int, err error)   { return 0, io.EOF }
func (c *MockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (c *MockConn) Close() error                       { c.closed = true; return nil }
func (c *MockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (c *MockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (c *MockConn) SetDeadline(t time.Time) error      { return nil }
func (c *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *MockConn) SetWriteDeadline(t time.Time) error { return nil }

// Common errors for testing
var (
	ErrConnectionRefused = errors.New("connection refused")
	ErrTimeout           = errors.New("i/o timeout")
	ErrCommandNotFound   = errors.New("command not found")
	ErrDNSLookupFailed   = errors.New("no such host")
	ErrFileNotFound      = errors.New("file not found")
	ErrPermissionDenied  = errors.New("permission denied")
)

// =============================================================================
// Filesystem Mocks
// =============================================================================

// MockFileSystemReader is a test implementation of FileSystemReader.
type MockFileSystemReader struct {
	// Responses maps path to the response
	Responses map[string]MockStatfsResponse
	// DefaultResponse is returned when no matching response is found
	DefaultResponse MockStatfsResponse
	// Calls records all paths requested
	Calls []string
}

// MockStatfsResponse represents a statfs response.
type MockStatfsResponse struct {
	Result *StatfsResult
	Error  error
}

// Statfs returns mock filesystem statistics.
func (m *MockFileSystemReader) Statfs(path string) (*StatfsResult, error) {
	m.Calls = append(m.Calls, path)
	if resp, ok := m.Responses[path]; ok {
		return resp.Result, resp.Error
	}
	return m.DefaultResponse.Result, m.DefaultResponse.Error
}

// NewMockFileSystemReader creates a new MockFileSystemReader with empty responses.
func NewMockFileSystemReader() *MockFileSystemReader {
	return &MockFileSystemReader{
		Responses: make(map[string]MockStatfsResponse),
		DefaultResponse: MockStatfsResponse{
			Result: &StatfsResult{
				Blocks: 1000000,
				Bfree:  500000,
				Bavail: 450000,
				Files:  100000,
				Ffree:  80000,
				Bsize:  4096,
			},
		},
	}
}

// MockProcReader is a test implementation of ProcReader.
type MockProcReader struct {
	// MemInfo is the memory info to return
	MemInfo *MemInfo
	// MemInfoError is the error to return from ReadMeminfo
	MemInfoError error
	// Processes is the list of processes to return
	Processes []ProcessInfo
	// ProcessesError is the error to return from ListProcesses
	ProcessesError error
	// Calls tracks which methods were called
	Calls []string
}

// ReadMeminfo returns mock memory information.
func (m *MockProcReader) ReadMeminfo() (*MemInfo, error) {
	m.Calls = append(m.Calls, "ReadMeminfo")
	if m.MemInfoError != nil {
		return nil, m.MemInfoError
	}
	if m.MemInfo != nil {
		return m.MemInfo, nil
	}
	// Default: 8GB swap, 2GB used
	return &MemInfo{
		SwapTotal: 8388608, // 8GB in KB
		SwapFree:  6291456, // 6GB in KB
	}, nil
}

// ListProcesses returns mock process information.
func (m *MockProcReader) ListProcesses() ([]ProcessInfo, error) {
	m.Calls = append(m.Calls, "ListProcesses")
	if m.ProcessesError != nil {
		return nil, m.ProcessesError
	}
	if m.Processes != nil {
		return m.Processes, nil
	}
	// Default: no processes
	return []ProcessInfo{}, nil
}

// NewMockProcReader creates a new MockProcReader.
func NewMockProcReader() *MockProcReader {
	return &MockProcReader{}
}

// MockPortReader is a test implementation of PortReader.
type MockPortReader struct {
	// PortInfo is the port info to return
	PortInfo *PortInfo
	// Error is the error to return
	Error error
	// Calls tracks method invocations
	Calls int
}

// ReadPortStats returns mock port statistics.
func (m *MockPortReader) ReadPortStats() (*PortInfo, error) {
	m.Calls++
	if m.Error != nil {
		return nil, m.Error
	}
	if m.PortInfo != nil {
		return m.PortInfo, nil
	}
	// Default: healthy port usage
	return &PortInfo{
		UsedPorts:   1000,
		TotalPorts:  28232,
		UsedPercent: 3,
		TimeWait:    50,
	}, nil
}

// NewMockPortReader creates a new MockPortReader.
func NewMockPortReader() *MockPortReader {
	return &MockPortReader{}
}

// MockCacheChecker is a test implementation of CacheChecker.
type MockCacheChecker struct {
	// Size is the cache size to return
	Size uint64
	// SizeError is the error to return from GetCacheSize
	SizeError error
	// CleanCount is the number of files cleaned
	CleanCount int
	// CleanSize is the bytes freed
	CleanSize uint64
	// CleanError is the error to return from CleanCache
	CleanError error
	// Calls tracks method invocations
	Calls []string
}

// GetCacheSize returns mock cache size.
func (m *MockCacheChecker) GetCacheSize(path string) (uint64, error) {
	m.Calls = append(m.Calls, "GetCacheSize:"+path)
	return m.Size, m.SizeError
}

// CleanCache returns mock cleanup results.
func (m *MockCacheChecker) CleanCache(path string, maxAge int) (int, uint64, error) {
	m.Calls = append(m.Calls, "CleanCache:"+path)
	return m.CleanCount, m.CleanSize, m.CleanError
}

// NewMockCacheChecker creates a new MockCacheChecker.
func NewMockCacheChecker() *MockCacheChecker {
	return &MockCacheChecker{}
}

// =============================================================================
// DNS Mocks
// =============================================================================

// DNSResolver abstracts DNS resolution for testability.
type DNSResolver interface {
	// LookupHost returns addresses for the given host
	LookupHost(host string) ([]string, error)
}

// MockDNSResolver is a test implementation of DNSResolver.
type MockDNSResolver struct {
	// Responses maps hostname to addresses
	Responses map[string][]string
	// Errors maps hostname to error
	Errors map[string]error
	// DefaultAddresses is returned when no matching response is found
	DefaultAddresses []string
	// DefaultError is returned when no matching error is found
	DefaultError error
	// Calls records all hostnames queried
	Calls []string
}

// LookupHost returns mock DNS results.
func (m *MockDNSResolver) LookupHost(host string) ([]string, error) {
	m.Calls = append(m.Calls, host)
	if err, ok := m.Errors[host]; ok && err != nil {
		return nil, err
	}
	if addrs, ok := m.Responses[host]; ok {
		return addrs, nil
	}
	if m.DefaultError != nil {
		return nil, m.DefaultError
	}
	if m.DefaultAddresses != nil {
		return m.DefaultAddresses, nil
	}
	return []string{"127.0.0.1"}, nil
}

// NewMockDNSResolver creates a new MockDNSResolver.
func NewMockDNSResolver() *MockDNSResolver {
	return &MockDNSResolver{
		Responses: make(map[string][]string),
		Errors:    make(map[string]error),
	}
}

// =============================================================================
// TLS Mocks
// =============================================================================

// CertInfo contains certificate information
type CertInfo struct {
	Subject    string
	Issuer     string
	NotBefore  time.Time
	NotAfter   time.Time
	DNSNames   []string
	IsValid    bool
	DaysUntil  int
}

// TLSDialer abstracts TLS connection for testability.
type TLSDialer interface {
	// GetCertificate returns certificate info for the given host:port
	GetCertificate(address string) (*CertInfo, error)
}

// MockTLSDialer is a test implementation of TLSDialer.
type MockTLSDialer struct {
	// Responses maps address to certificate info
	Responses map[string]*CertInfo
	// Errors maps address to error
	Errors map[string]error
	// DefaultCert is returned when no matching response is found
	DefaultCert *CertInfo
	// DefaultError is returned when no matching error is found
	DefaultError error
	// Calls records all addresses queried
	Calls []string
}

// GetCertificate returns mock certificate info.
func (m *MockTLSDialer) GetCertificate(address string) (*CertInfo, error) {
	m.Calls = append(m.Calls, address)
	if err, ok := m.Errors[address]; ok && err != nil {
		return nil, err
	}
	if cert, ok := m.Responses[address]; ok {
		return cert, nil
	}
	if m.DefaultError != nil {
		return nil, m.DefaultError
	}
	if m.DefaultCert != nil {
		return m.DefaultCert, nil
	}
	// Default: valid cert expiring in 90 days
	return &CertInfo{
		Subject:   "example.com",
		Issuer:    "Let's Encrypt",
		NotBefore: time.Now().Add(-30 * 24 * time.Hour),
		NotAfter:  time.Now().Add(90 * 24 * time.Hour),
		DNSNames:  []string{"example.com", "www.example.com"},
		IsValid:   true,
		DaysUntil: 90,
	}, nil
}

// NewMockTLSDialer creates a new MockTLSDialer.
func NewMockTLSDialer() *MockTLSDialer {
	return &MockTLSDialer{
		Responses: make(map[string]*CertInfo),
		Errors:    make(map[string]error),
	}
}
