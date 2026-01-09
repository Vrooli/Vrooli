// Package checks tests for mock implementations
// [REQ:TEST-SEAM-001] Verify mock implementations work correctly
package checks

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// =============================================================================
// MockExecutor Tests
// =============================================================================

// TestNewMockExecutor verifies mock executor initialization
func TestNewMockExecutor(t *testing.T) {
	exec := NewMockExecutor()

	if exec == nil {
		t.Fatal("NewMockExecutor() returned nil")
	}
	if exec.Responses == nil {
		t.Error("Responses map not initialized")
	}
	if len(exec.Calls) != 0 {
		t.Error("Calls should be empty initially")
	}
}

// TestMockExecutorOutput verifies Output method
func TestMockExecutorOutput(t *testing.T) {
	exec := NewMockExecutor()
	exec.Responses["echo hello"] = MockResponse{
		Output: []byte("hello\n"),
		Error:  nil,
	}

	ctx := context.Background()
	output, err := exec.Output(ctx, "echo", "hello")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(output) != "hello\n" {
		t.Errorf("Output = %q, want %q", output, "hello\n")
	}
	if len(exec.Calls) != 1 {
		t.Errorf("Calls = %d, want 1", len(exec.Calls))
	}
	if exec.Calls[0].Name != "echo" {
		t.Errorf("Call name = %q, want %q", exec.Calls[0].Name, "echo")
	}
}

// TestMockExecutorOutputDefault verifies default response
func TestMockExecutorOutputDefault(t *testing.T) {
	exec := NewMockExecutor()
	exec.DefaultResponse = MockResponse{
		Output: []byte("default"),
		Error:  nil,
	}

	ctx := context.Background()
	output, err := exec.Output(ctx, "unknown", "command")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(output) != "default" {
		t.Errorf("Output = %q, want %q", output, "default")
	}
}

// TestMockExecutorOutputError verifies error handling
func TestMockExecutorOutputError(t *testing.T) {
	exec := NewMockExecutor()
	exec.Responses["fail cmd"] = MockResponse{
		Output: nil,
		Error:  ErrCommandNotFound,
	}

	ctx := context.Background()
	_, err := exec.Output(ctx, "fail", "cmd")

	if err != ErrCommandNotFound {
		t.Errorf("error = %v, want %v", err, ErrCommandNotFound)
	}
}

// TestMockExecutorCombinedOutput verifies CombinedOutput delegates to Output
func TestMockExecutorCombinedOutput(t *testing.T) {
	exec := NewMockExecutor()
	exec.Responses["test cmd"] = MockResponse{
		Output: []byte("combined"),
		Error:  nil,
	}

	ctx := context.Background()
	output, err := exec.CombinedOutput(ctx, "test", "cmd")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(output) != "combined" {
		t.Errorf("CombinedOutput = %q, want %q", output, "combined")
	}
}

// TestMockExecutorRun verifies Run method
func TestMockExecutorRun(t *testing.T) {
	exec := NewMockExecutor()
	exec.Responses["run cmd"] = MockResponse{
		Output: nil,
		Error:  nil,
	}

	ctx := context.Background()
	err := exec.Run(ctx, "run", "cmd")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(exec.Calls) != 1 {
		t.Errorf("Calls = %d, want 1", len(exec.Calls))
	}
}

// TestMockExecutorRunError verifies Run error handling
func TestMockExecutorRunError(t *testing.T) {
	exec := NewMockExecutor()
	exec.Responses["fail run"] = MockResponse{
		Error: ErrPermissionDenied,
	}

	ctx := context.Background()
	err := exec.Run(ctx, "fail", "run")

	if err != ErrPermissionDenied {
		t.Errorf("error = %v, want %v", err, ErrPermissionDenied)
	}
}

// TestCommandKey verifies command key generation
func TestCommandKey(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		args []string
		want string
	}{
		{"no args", "echo", nil, "echo"},
		{"one arg", "echo", []string{"hello"}, "echo hello"},
		{"multiple args", "ls", []string{"-la", "/tmp"}, "ls -la /tmp"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := commandKey(tc.cmd, tc.args)
			if got != tc.want {
				t.Errorf("commandKey(%q, %v) = %q, want %q", tc.cmd, tc.args, got, tc.want)
			}
		})
	}
}

// =============================================================================
// MockHTTPClient Tests
// =============================================================================

// TestNewMockHTTPClient verifies mock HTTP client initialization
func TestNewMockHTTPClient(t *testing.T) {
	client := NewMockHTTPClient()

	if client == nil {
		t.Fatal("NewMockHTTPClient() returned nil")
	}
	if client.Responses == nil {
		t.Error("Responses map not initialized")
	}
	if client.DefaultResponse.StatusCode != 200 {
		t.Errorf("DefaultResponse.StatusCode = %d, want 200", client.DefaultResponse.StatusCode)
	}
}

// TestMockHTTPClientDo verifies Do method with configured response
func TestMockHTTPClientDo(t *testing.T) {
	client := NewMockHTTPClient()
	client.Responses["http://example.com/api"] = MockHTTPResponse{
		StatusCode: 201,
		Body:       `{"created":true}`,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}

	req, _ := http.NewRequest("GET", "http://example.com/api", nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Errorf("StatusCode = %d, want 201", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want %q", resp.Header.Get("Content-Type"), "application/json")
	}
	if len(client.Calls) != 1 {
		t.Errorf("Calls = %d, want 1", len(client.Calls))
	}
}

// TestMockHTTPClientDoDefault verifies default response
func TestMockHTTPClientDoDefault(t *testing.T) {
	client := NewMockHTTPClient()
	client.DefaultResponse = MockHTTPResponse{
		StatusCode: 200,
		Body:       "default",
	}

	req, _ := http.NewRequest("GET", "http://unknown.com", nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

// TestMockHTTPClientDoError verifies error handling
func TestMockHTTPClientDoError(t *testing.T) {
	client := NewMockHTTPClient()
	client.Responses["http://fail.com"] = MockHTTPResponse{
		Error: ErrConnectionRefused,
	}

	req, _ := http.NewRequest("GET", "http://fail.com", nil)
	_, err := client.Do(req)

	if err != ErrConnectionRefused {
		t.Errorf("error = %v, want %v", err, ErrConnectionRefused)
	}
}

// =============================================================================
// MockDialer Tests
// =============================================================================

// TestNewMockDialer verifies mock dialer initialization
func TestNewMockDialer(t *testing.T) {
	dialer := NewMockDialer()

	if dialer == nil {
		t.Fatal("NewMockDialer() returned nil")
	}
	if dialer.Responses == nil {
		t.Error("Responses map not initialized")
	}
}

// TestMockDialerDialTimeout verifies DialTimeout with configured response
func TestMockDialerDialTimeout(t *testing.T) {
	dialer := NewMockDialer()
	mockConn := &MockConn{}
	dialer.Responses["127.0.0.1:8080"] = MockDialResponse{
		Conn:  mockConn,
		Error: nil,
	}

	conn, err := dialer.DialTimeout("tcp", "127.0.0.1:8080", 5*time.Second)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if conn != mockConn {
		t.Error("expected mockConn to be returned")
	}
	if len(dialer.Calls) != 1 {
		t.Errorf("Calls = %d, want 1", len(dialer.Calls))
	}
}

// TestMockDialerDialTimeoutError verifies error handling
func TestMockDialerDialTimeoutError(t *testing.T) {
	dialer := NewMockDialer()
	dialer.Responses["unreachable:80"] = MockDialResponse{
		Error: ErrTimeout,
	}

	_, err := dialer.DialTimeout("tcp", "unreachable:80", 5*time.Second)

	if err != ErrTimeout {
		t.Errorf("error = %v, want %v", err, ErrTimeout)
	}
}

// TestMockDialerDefault verifies default response
func TestMockDialerDefault(t *testing.T) {
	dialer := NewMockDialer()
	mockConn := &MockConn{}
	dialer.DefaultResponse = MockDialResponse{
		Conn: mockConn,
	}

	conn, err := dialer.DialTimeout("tcp", "any.addr:80", 5*time.Second)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if conn != mockConn {
		t.Error("expected default mockConn")
	}
}

// =============================================================================
// MockConn Tests
// =============================================================================

// TestMockConnInterface verifies MockConn implements net.Conn
func TestMockConnInterface(t *testing.T) {
	conn := &MockConn{}

	// Test Read
	buf := make([]byte, 10)
	n, err := conn.Read(buf)
	if n != 0 {
		t.Errorf("Read n = %d, want 0", n)
	}
	if err == nil {
		t.Error("Read should return EOF")
	}

	// Test Write
	n, err = conn.Write([]byte("hello"))
	if n != 5 {
		t.Errorf("Write n = %d, want 5", n)
	}
	if err != nil {
		t.Errorf("Write error = %v, want nil", err)
	}

	// Test Close
	err = conn.Close()
	if err != nil {
		t.Errorf("Close error = %v, want nil", err)
	}
	if !conn.closed {
		t.Error("Close should set closed = true")
	}

	// Test address methods
	if conn.LocalAddr() == nil {
		t.Error("LocalAddr() should not return nil")
	}
	if conn.RemoteAddr() == nil {
		t.Error("RemoteAddr() should not return nil")
	}

	// Test deadline methods
	now := time.Now()
	if err := conn.SetDeadline(now); err != nil {
		t.Errorf("SetDeadline error = %v", err)
	}
	if err := conn.SetReadDeadline(now); err != nil {
		t.Errorf("SetReadDeadline error = %v", err)
	}
	if err := conn.SetWriteDeadline(now); err != nil {
		t.Errorf("SetWriteDeadline error = %v", err)
	}
}

// =============================================================================
// MockFileSystemReader Tests
// =============================================================================

// TestNewMockFileSystemReader verifies initialization
func TestNewMockFileSystemReader(t *testing.T) {
	reader := NewMockFileSystemReader()

	if reader == nil {
		t.Fatal("NewMockFileSystemReader() returned nil")
	}
	if reader.Responses == nil {
		t.Error("Responses map not initialized")
	}
	if reader.DefaultResponse.Result == nil {
		t.Error("DefaultResponse.Result should be set")
	}
}

// TestMockFileSystemReaderStatfs verifies Statfs method
func TestMockFileSystemReaderStatfs(t *testing.T) {
	reader := NewMockFileSystemReader()
	reader.Responses["/custom/path"] = MockStatfsResponse{
		Result: &StatfsResult{
			Blocks: 2000000,
			Bfree:  1000000,
		},
	}

	result, err := reader.Statfs("/custom/path")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Blocks != 2000000 {
		t.Errorf("Blocks = %d, want 2000000", result.Blocks)
	}
	if len(reader.Calls) != 1 {
		t.Errorf("Calls = %d, want 1", len(reader.Calls))
	}
}

// TestMockFileSystemReaderStatfsDefault verifies default response
func TestMockFileSystemReaderStatfsDefault(t *testing.T) {
	reader := NewMockFileSystemReader()

	result, err := reader.Statfs("/any/path")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Blocks != 1000000 {
		t.Errorf("Blocks = %d, want 1000000", result.Blocks)
	}
}

// TestMockFileSystemReaderStatfsError verifies error handling
func TestMockFileSystemReaderStatfsError(t *testing.T) {
	reader := NewMockFileSystemReader()
	reader.Responses["/error/path"] = MockStatfsResponse{
		Error: ErrFileNotFound,
	}

	_, err := reader.Statfs("/error/path")

	if err != ErrFileNotFound {
		t.Errorf("error = %v, want %v", err, ErrFileNotFound)
	}
}

// =============================================================================
// MockProcReader Tests
// =============================================================================

// TestNewMockProcReader verifies initialization
func TestNewMockProcReader(t *testing.T) {
	reader := NewMockProcReader()

	if reader == nil {
		t.Fatal("NewMockProcReader() returned nil")
	}
}

// TestMockProcReaderReadMeminfo verifies ReadMeminfo method
func TestMockProcReaderReadMeminfo(t *testing.T) {
	reader := NewMockProcReader()
	reader.MemInfo = &MemInfo{
		SwapTotal: 16000000,
		SwapFree:  12000000,
	}

	info, err := reader.ReadMeminfo()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info.SwapTotal != 16000000 {
		t.Errorf("SwapTotal = %d, want 16000000", info.SwapTotal)
	}
	if len(reader.Calls) != 1 || reader.Calls[0] != "ReadMeminfo" {
		t.Errorf("Calls = %v, want [ReadMeminfo]", reader.Calls)
	}
}

// TestMockProcReaderReadMeminfoDefault verifies default response
func TestMockProcReaderReadMeminfoDefault(t *testing.T) {
	reader := NewMockProcReader()

	info, err := reader.ReadMeminfo()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info.SwapTotal != 8388608 {
		t.Errorf("SwapTotal = %d, want 8388608 (default)", info.SwapTotal)
	}
}

// TestMockProcReaderReadMeminfoError verifies error handling
func TestMockProcReaderReadMeminfoError(t *testing.T) {
	reader := NewMockProcReader()
	reader.MemInfoError = ErrFileNotFound

	_, err := reader.ReadMeminfo()

	if err != ErrFileNotFound {
		t.Errorf("error = %v, want %v", err, ErrFileNotFound)
	}
}

// TestMockProcReaderListProcesses verifies ListProcesses method
func TestMockProcReaderListProcesses(t *testing.T) {
	reader := NewMockProcReader()
	reader.Processes = []ProcessInfo{
		{PID: 1, State: "S", Comm: "init"},
		{PID: 2, State: "Z", Comm: "zombie"},
	}

	procs, err := reader.ListProcesses()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(procs) != 2 {
		t.Errorf("len(procs) = %d, want 2", len(procs))
	}
	if procs[1].State != "Z" {
		t.Errorf("procs[1].State = %q, want %q", procs[1].State, "Z")
	}
}

// TestMockProcReaderListProcessesDefault verifies default response
func TestMockProcReaderListProcessesDefault(t *testing.T) {
	reader := NewMockProcReader()

	procs, err := reader.ListProcesses()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(procs) != 0 {
		t.Errorf("len(procs) = %d, want 0 (default)", len(procs))
	}
}

// TestMockProcReaderListProcessesError verifies error handling
func TestMockProcReaderListProcessesError(t *testing.T) {
	reader := NewMockProcReader()
	reader.ProcessesError = ErrPermissionDenied

	_, err := reader.ListProcesses()

	if err != ErrPermissionDenied {
		t.Errorf("error = %v, want %v", err, ErrPermissionDenied)
	}
}

// =============================================================================
// MockPortReader Tests
// =============================================================================

// TestNewMockPortReader verifies initialization
func TestNewMockPortReader(t *testing.T) {
	reader := NewMockPortReader()

	if reader == nil {
		t.Fatal("NewMockPortReader() returned nil")
	}
}

// TestMockPortReaderReadPortStats verifies ReadPortStats method
func TestMockPortReaderReadPortStats(t *testing.T) {
	reader := NewMockPortReader()
	reader.PortInfo = &PortInfo{
		UsedPorts:   5000,
		TotalPorts:  30000,
		UsedPercent: 16,
		TimeWait:    200,
	}

	info, err := reader.ReadPortStats()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info.UsedPorts != 5000 {
		t.Errorf("UsedPorts = %d, want 5000", info.UsedPorts)
	}
	if reader.Calls != 1 {
		t.Errorf("Calls = %d, want 1", reader.Calls)
	}
}

// TestMockPortReaderReadPortStatsDefault verifies default response
func TestMockPortReaderReadPortStatsDefault(t *testing.T) {
	reader := NewMockPortReader()

	info, err := reader.ReadPortStats()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if info.UsedPorts != 1000 {
		t.Errorf("UsedPorts = %d, want 1000 (default)", info.UsedPorts)
	}
}

// TestMockPortReaderReadPortStatsError verifies error handling
func TestMockPortReaderReadPortStatsError(t *testing.T) {
	reader := NewMockPortReader()
	reader.Error = ErrFileNotFound

	_, err := reader.ReadPortStats()

	if err != ErrFileNotFound {
		t.Errorf("error = %v, want %v", err, ErrFileNotFound)
	}
}

// =============================================================================
// MockCacheChecker Tests
// =============================================================================

// TestNewMockCacheChecker verifies initialization
func TestNewMockCacheChecker(t *testing.T) {
	checker := NewMockCacheChecker()

	if checker == nil {
		t.Fatal("NewMockCacheChecker() returned nil")
	}
}

// TestMockCacheCheckerGetCacheSize verifies GetCacheSize method
func TestMockCacheCheckerGetCacheSize(t *testing.T) {
	checker := NewMockCacheChecker()
	checker.Size = 1024 * 1024 // 1MB

	size, err := checker.GetCacheSize("/path/to/cache")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if size != 1024*1024 {
		t.Errorf("size = %d, want 1048576", size)
	}
	if len(checker.Calls) != 1 || checker.Calls[0] != "GetCacheSize:/path/to/cache" {
		t.Errorf("Calls = %v, want [GetCacheSize:/path/to/cache]", checker.Calls)
	}
}

// TestMockCacheCheckerGetCacheSizeError verifies error handling
func TestMockCacheCheckerGetCacheSizeError(t *testing.T) {
	checker := NewMockCacheChecker()
	checker.SizeError = ErrPermissionDenied

	_, err := checker.GetCacheSize("/path")

	if err != ErrPermissionDenied {
		t.Errorf("error = %v, want %v", err, ErrPermissionDenied)
	}
}

// TestMockCacheCheckerCleanCache verifies CleanCache method
func TestMockCacheCheckerCleanCache(t *testing.T) {
	checker := NewMockCacheChecker()
	checker.CleanCount = 10
	checker.CleanSize = 5000

	count, size, err := checker.CleanCache("/path/to/cache", 30)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if count != 10 {
		t.Errorf("count = %d, want 10", count)
	}
	if size != 5000 {
		t.Errorf("size = %d, want 5000", size)
	}
}

// TestMockCacheCheckerCleanCacheError verifies error handling
func TestMockCacheCheckerCleanCacheError(t *testing.T) {
	checker := NewMockCacheChecker()
	checker.CleanError = ErrPermissionDenied

	_, _, err := checker.CleanCache("/path", 30)

	if err != ErrPermissionDenied {
		t.Errorf("error = %v, want %v", err, ErrPermissionDenied)
	}
}

// =============================================================================
// MockDNSResolver Tests
// =============================================================================

// TestNewMockDNSResolver verifies initialization
func TestNewMockDNSResolver(t *testing.T) {
	resolver := NewMockDNSResolver()

	if resolver == nil {
		t.Fatal("NewMockDNSResolver() returned nil")
	}
	if resolver.Responses == nil {
		t.Error("Responses map not initialized")
	}
	if resolver.Errors == nil {
		t.Error("Errors map not initialized")
	}
}

// TestMockDNSResolverLookupHost verifies LookupHost method
func TestMockDNSResolverLookupHost(t *testing.T) {
	resolver := NewMockDNSResolver()
	resolver.Responses["example.com"] = []string{"192.168.1.1", "192.168.1.2"}

	addrs, err := resolver.LookupHost("example.com")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(addrs) != 2 {
		t.Errorf("len(addrs) = %d, want 2", len(addrs))
	}
	if len(resolver.Calls) != 1 || resolver.Calls[0] != "example.com" {
		t.Errorf("Calls = %v, want [example.com]", resolver.Calls)
	}
}

// TestMockDNSResolverLookupHostError verifies error handling
func TestMockDNSResolverLookupHostError(t *testing.T) {
	resolver := NewMockDNSResolver()
	resolver.Errors["bad.host"] = ErrDNSLookupFailed

	_, err := resolver.LookupHost("bad.host")

	if err != ErrDNSLookupFailed {
		t.Errorf("error = %v, want %v", err, ErrDNSLookupFailed)
	}
}

// TestMockDNSResolverLookupHostDefault verifies default response
func TestMockDNSResolverLookupHostDefault(t *testing.T) {
	resolver := NewMockDNSResolver()
	resolver.DefaultAddresses = []string{"8.8.8.8"}

	addrs, err := resolver.LookupHost("any.host")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
		t.Errorf("addrs = %v, want [8.8.8.8]", addrs)
	}
}

// TestMockDNSResolverLookupHostDefaultFallback verifies final fallback
func TestMockDNSResolverLookupHostDefaultFallback(t *testing.T) {
	resolver := NewMockDNSResolver()
	// No configured response, no default addresses

	addrs, err := resolver.LookupHost("any.host")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(addrs) != 1 || addrs[0] != "127.0.0.1" {
		t.Errorf("addrs = %v, want [127.0.0.1]", addrs)
	}
}

// TestMockDNSResolverLookupHostDefaultError verifies default error
func TestMockDNSResolverLookupHostDefaultError(t *testing.T) {
	resolver := NewMockDNSResolver()
	resolver.DefaultError = ErrTimeout

	_, err := resolver.LookupHost("any.host")

	if err != ErrTimeout {
		t.Errorf("error = %v, want %v", err, ErrTimeout)
	}
}

// =============================================================================
// MockTLSDialer Tests
// =============================================================================

// TestNewMockTLSDialer verifies initialization
func TestNewMockTLSDialer(t *testing.T) {
	dialer := NewMockTLSDialer()

	if dialer == nil {
		t.Fatal("NewMockTLSDialer() returned nil")
	}
	if dialer.Responses == nil {
		t.Error("Responses map not initialized")
	}
	if dialer.Errors == nil {
		t.Error("Errors map not initialized")
	}
}

// TestMockTLSDialerGetCertificate verifies GetCertificate method
func TestMockTLSDialerGetCertificate(t *testing.T) {
	dialer := NewMockTLSDialer()
	dialer.Responses["example.com:443"] = &CertInfo{
		Subject:   "example.com",
		Issuer:    "Test CA",
		IsValid:   true,
		DaysUntil: 30,
	}

	cert, err := dialer.GetCertificate("example.com:443")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cert.Subject != "example.com" {
		t.Errorf("Subject = %q, want %q", cert.Subject, "example.com")
	}
	if len(dialer.Calls) != 1 || dialer.Calls[0] != "example.com:443" {
		t.Errorf("Calls = %v, want [example.com:443]", dialer.Calls)
	}
}

// TestMockTLSDialerGetCertificateError verifies error handling
func TestMockTLSDialerGetCertificateError(t *testing.T) {
	dialer := NewMockTLSDialer()
	dialer.Errors["bad.host:443"] = ErrConnectionRefused

	_, err := dialer.GetCertificate("bad.host:443")

	if err != ErrConnectionRefused {
		t.Errorf("error = %v, want %v", err, ErrConnectionRefused)
	}
}

// TestMockTLSDialerGetCertificateDefault verifies default response
func TestMockTLSDialerGetCertificateDefault(t *testing.T) {
	dialer := NewMockTLSDialer()
	dialer.DefaultCert = &CertInfo{
		Subject:   "default.com",
		DaysUntil: 60,
	}

	cert, err := dialer.GetCertificate("any.host:443")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cert.Subject != "default.com" {
		t.Errorf("Subject = %q, want %q", cert.Subject, "default.com")
	}
}

// TestMockTLSDialerGetCertificateFallback verifies final fallback
func TestMockTLSDialerGetCertificateFallback(t *testing.T) {
	dialer := NewMockTLSDialer()
	// No configured response, no default cert

	cert, err := dialer.GetCertificate("any.host:443")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cert.Subject != "example.com" {
		t.Errorf("Subject = %q, want %q (fallback)", cert.Subject, "example.com")
	}
	if cert.DaysUntil != 90 {
		t.Errorf("DaysUntil = %d, want 90 (fallback)", cert.DaysUntil)
	}
}

// TestMockTLSDialerGetCertificateDefaultError verifies default error
func TestMockTLSDialerGetCertificateDefaultError(t *testing.T) {
	dialer := NewMockTLSDialer()
	dialer.DefaultError = ErrTimeout

	_, err := dialer.GetCertificate("any.host:443")

	if err != ErrTimeout {
		t.Errorf("error = %v, want %v", err, ErrTimeout)
	}
}

// =============================================================================
// Common Error Variables Tests
// =============================================================================

// TestCommonErrors verifies error variables are defined
func TestCommonErrors(t *testing.T) {
	errors := map[string]error{
		"ErrConnectionRefused": ErrConnectionRefused,
		"ErrTimeout":           ErrTimeout,
		"ErrCommandNotFound":   ErrCommandNotFound,
		"ErrDNSLookupFailed":   ErrDNSLookupFailed,
		"ErrFileNotFound":      ErrFileNotFound,
		"ErrPermissionDenied":  ErrPermissionDenied,
	}

	for name, err := range errors {
		if err == nil {
			t.Errorf("%s should not be nil", name)
		}
		if err.Error() == "" {
			t.Errorf("%s should have non-empty message", name)
		}
	}
}
