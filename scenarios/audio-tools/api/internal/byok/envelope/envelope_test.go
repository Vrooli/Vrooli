package envelope

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
)

func TestFromHTTP(t *testing.T) {
	mk := func(p, k, l, u string) http.Header {
		h := http.Header{}
		if p != "" {
			h.Set(HeaderProvider, p)
		}
		if k != "" {
			h.Set(HeaderBYOKKey, k)
		}
		if l != "" {
			h.Set(HeaderLPBSToken, l)
		}
		if u != "" {
			h.Set(HeaderUserIdentity, u)
		}
		return h
	}
	tests := []struct {
		name string
		hdr  http.Header
		want Envelope
	}{
		{
			name: "all four fields populated",
			hdr:  mk("openai-whisper", "sk-abc", "lpbs-tok", "user-42"),
			want: Envelope{
				Provider:     "openai-whisper",
				Key:          "sk-abc",
				LPBSToken:    "lpbs-tok",
				UserIdentity: "user-42",
			},
		},
		{
			name: "all empty",
			hdr:  http.Header{},
			want: Envelope{},
		},
		{
			name: "trims surrounding whitespace",
			hdr:  mk("  deepgram  ", "\tsecret\n", "  ", " who "),
			want: Envelope{
				Provider:     "deepgram",
				Key:          "secret",
				LPBSToken:    "",
				UserIdentity: "who",
			},
		},
		{
			name: "lookup is case-insensitive on stored canonical key",
			hdr: func() http.Header {
				h := http.Header{}
				h["X-Audio-Byok-Provider"] = []string{"elevenlabs"}
				h["X-Audio-Byok-Key"] = []string{"k"}
				return h
			}(),
			want: Envelope{Provider: "elevenlabs", Key: "k"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FromHTTP(tc.hdr)
			if got != tc.want {
				t.Fatalf("FromHTTP = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestFromConnectRequest pins the Connect-RPC adapter to the underlying
// FromHTTP behavior.
func TestFromConnectRequest(t *testing.T) {
	req := connect.NewRequest(&struct{}{})
	req.Header().Set(HeaderProvider, "openrouter")
	req.Header().Set(HeaderBYOKKey, "sk-from-connect")
	req.Header().Set(HeaderLPBSToken, "tok")
	req.Header().Set(HeaderUserIdentity, "user-7")
	got := FromConnectRequest(req)
	want := Envelope{Provider: "openrouter", Key: "sk-from-connect", LPBSToken: "tok", UserIdentity: "user-7"}
	if got != want {
		t.Fatalf("FromConnectRequest = %+v, want %+v", got, want)
	}
}

// TestFromConnectStream pins the streaming adapter to FromHTTP. Uses a
// raw httptest.NewRequest header rather than a real bidi stream because
// the function only reads the header.
func TestFromConnectStream(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(HeaderProvider, "elevenlabs")
	req.Header.Set(HeaderUserIdentity, "u-1")
	got := FromConnectStream(req.Header)
	want := Envelope{Provider: "elevenlabs", UserIdentity: "u-1"}
	if got != want {
		t.Fatalf("FromConnectStream = %+v, want %+v", got, want)
	}
}

func TestHeaderConstantsMatchWireContract(t *testing.T) {
	cases := map[string]string{
		HeaderProvider:     "X-Audio-BYOK-Provider",
		HeaderKey:          "X-Audio-BYOK-Key",
		HeaderLPBSToken:    "X-Audio-LPBS-Token",
		HeaderUserIdentity: "X-Audio-User-Identity",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("header constant changed: got %q want %q", got, want)
		}
	}
}
