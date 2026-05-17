// Package envelope centralizes the wire-side BYOK request envelope.
//
// Audio-tools handlers accept BYOK credentials and operator metadata as a
// fixed set of HTTP headers carried on Connect-RPC, multipart, and bidi
// streaming requests. The header names are part of the wire contract and
// must match across all transports. This package owns the canonical header
// names and the typed Envelope struct that handlers consume.
package envelope

import (
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

// Wire header names. These must match byte-for-byte across every audio-tools
// transport — Connect unary, Connect bidi, and multipart REST.
const (
	HeaderProvider     = "X-Audio-BYOK-Provider"
	HeaderKey          = "X-Audio-BYOK-Key"
	HeaderLPBSToken    = "X-Audio-LPBS-Token"
	HeaderUserIdentity = "X-Audio-User-Identity"
)

// Envelope is the per-request operator/BYOK context parsed from the wire
// headers. All fields are optional and zero-valued when absent.
type Envelope struct {
	Provider     string
	Key          string
	LPBSToken    string
	UserIdentity string
}

// FromHTTP parses the envelope from raw HTTP headers. Surrounding whitespace
// on each value is trimmed.
func FromHTTP(h http.Header) Envelope {
	return Envelope{
		Provider:     strings.TrimSpace(h.Get(HeaderProvider)),
		Key:          strings.TrimSpace(h.Get(HeaderKey)),
		LPBSToken:    strings.TrimSpace(h.Get(HeaderLPBSToken)),
		UserIdentity: strings.TrimSpace(h.Get(HeaderUserIdentity)),
	}
}

// FromConnectRequest parses the envelope from a Connect unary request.
func FromConnectRequest(req connect.AnyRequest) Envelope {
	return FromHTTP(req.Header())
}

// FromConnectStream parses the envelope from a Connect streaming handler's
// per-stream request header (bidi or client-streaming).
func FromConnectStream(h http.Header) Envelope {
	return FromHTTP(h)
}
