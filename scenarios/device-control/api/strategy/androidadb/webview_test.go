package androidadb

import (
	"context"
	"errors"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestAttachWebViewMatchesPackageProcessAndAllocatesCDPForward(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell pidof com.example.hello":                            []byte("4321\n"),
		"adb -s serial-1 shell cat /proc/net/unix":                                 []byte("Num RefCount Protocol Flags Type St Inode Path\n0001 2 0 10000 1 01 7 @webview_devtools_remote_4321\n"),
		"adb -s serial-1 forward tcp:0 localabstract:webview_devtools_remote_4321": []byte("tcp:43123\n"),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	adapter.rendererID = func(context.Context, string) (string, error) { return "renderer-1", nil }
	endpoint, err := adapter.AttachWebView(context.Background(), "com.example.hello")
	require.NoError(t, err)
	require.Equal(t, strategy.WebViewEndpoint{Package: "com.example.hello", Socket: "webview_devtools_remote_4321", CDPEndpoint: "http://127.0.0.1:43123", RendererID: "renderer-1", Transport: "adb-forward"}, endpoint)
}

func TestAttachWebViewClassifiesPortForwardFailure(t *testing.T) {
	forwardErr := errors.New("forward timed out with device transport details")
	runner := &scriptedRunner{
		responses: map[string][]byte{
			"adb -s serial-1 shell pidof com.example.hello": []byte("4321\n"),
			"adb -s serial-1 shell cat /proc/net/unix":      []byte("Num RefCount Protocol Flags Type St Inode Path\n0001 2 0 10000 1 01 7 @webview_devtools_remote_4321\n"),
		},
		errors: map[string]error{
			"adb -s serial-1 forward tcp:0 localabstract:webview_devtools_remote_4321": forwardErr,
		},
	}
	_, err := NewWithRunner(runner, "serial-1").AttachWebView(context.Background(), "com.example.hello")
	var failure *strategy.WebViewAttachFailure
	require.ErrorAs(t, err, &failure)
	require.Equal(t, strategy.WebViewStagePortForward, failure.Stage)
	require.ErrorIs(t, err, forwardErr)
}

func TestWebViewSocketMatchingRejectsDifferentProcess(t *testing.T) {
	_, err := webViewSocketForPID([]byte("0001 2 0 10000 1 01 7 @webview_devtools_remote_9876\n"), "4321")
	require.ErrorContains(t, err, "matched process 4321")
}
