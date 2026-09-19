package audioports

import (
	"context"

	"connectrpc.com/connect"

	"swarm-manager/integrations/audiotools"
)

// remoteBase holds the Client + Credentials fields and the three shared
// helpers (ensure, handleErr, attach) used by every Remote*Admin adapter.
// Embed it in a Remote* struct to inherit these methods.
type remoteBase struct {
	Client      *audiotools.Client
	Credentials func(ctx context.Context) audiotools.Credentials
}

// NewRemoteStreamConfigAdmin builds a StreamConfigAdmin backed by the given audio-tools client.
func NewRemoteStreamConfigAdmin(client *audiotools.Client) *RemoteStreamConfigAdmin {
	return &RemoteStreamConfigAdmin{remoteBase{Client: client}}
}

// NewRemoteWakeWordAdmin builds a WakeWordAdmin backed by the given audio-tools client.
func NewRemoteWakeWordAdmin(client *audiotools.Client) *RemoteWakeWordAdmin {
	return &RemoteWakeWordAdmin{remoteBase{Client: client}}
}

// NewRemoteSpeakerAdmin builds a SpeakerAdmin backed by the given audio-tools client.
func NewRemoteSpeakerAdmin(client *audiotools.Client) *RemoteSpeakerAdmin {
	return &RemoteSpeakerAdmin{remoteBase{Client: client}}
}

// NewRemoteTTSConfigAdmin builds a TTSConfigAdmin backed by the given audio-tools client.
func NewRemoteTTSConfigAdmin(client *audiotools.Client) *RemoteTTSConfigAdmin {
	return &RemoteTTSConfigAdmin{remoteBase{Client: client}}
}

// NewRemoteSummarizeConfigAdmin builds a SummarizeConfigAdmin backed by the given audio-tools client.
func NewRemoteSummarizeConfigAdmin(client *audiotools.Client) *RemoteSummarizeConfigAdmin {
	return &RemoteSummarizeConfigAdmin{remoteBase{Client: client}}
}

func (r *remoteBase) ensure() error {
	if r == nil || r.Client == nil {
		return audiotools.ErrUnavailable
	}
	if err := r.Client.Ensure(); err != nil {
		return audiotools.ErrUnavailable
	}
	return nil
}

func (r *remoteBase) handleErr(err error) error {
	if err == nil {
		return nil
	}
	if isTransportFailure(err) {
		r.Client.HandleTransportFailure()
	}
	return audiotools.NormalizeError(err)
}

func (r *remoteBase) attach(ctx context.Context, req connect.AnyRequest) {
	if r.Credentials == nil || req == nil {
		return
	}
	creds := r.Credentials(ctx)
	if creds.BYOKKey != "" {
		req.Header().Set("X-Audio-BYOK-Key", creds.BYOKKey)
		req.Header().Set("X-Audio-BYOK-Provider", creds.BYOKProvider)
	}
	if creds.LPBSToken != "" {
		req.Header().Set("X-Audio-LPBS-Token", creds.LPBSToken)
	}
	if creds.UserIdentity != "" {
		req.Header().Set("X-Audio-User-Identity", creds.UserIdentity)
	}
}
