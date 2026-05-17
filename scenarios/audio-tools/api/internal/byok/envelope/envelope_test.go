package envelope

import (
	"net/http"
	"testing"
)

func TestFromHTTP(t *testing.T) {
	mk := func(p, k, l, u string) http.Header {
		h := http.Header{}
		if p != "" {
			h.Set(HeaderProvider, p)
		}
		if k != "" {
			h.Set(HeaderKey, k)
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
