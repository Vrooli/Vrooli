package conversations

import (
	"net/http"
	"time"

	"notification-hub/internal/hub"
	"notification-hub/internal/identity"
	"notification-hub/internal/module"
)

// ModuleWithUploadLimits is the module with a smaller upload capacity and a
// shorter wait for a slot, so a test can fill the slots without a real upload.
func ModuleWithUploadLimits(service *hub.Service, verifier identity.Verifier, slots int, wait time.Duration) module.Module {
	h := newHandler(service, verifier, identity.ServiceVerifiers{}, nil)
	h.uploads = make(chan struct{}, slots)
	h.uploadWait = wait
	return h.module()
}

// GateUploads is the upload gate in front of next, with its own slots, so a
// test can hold them with a handler of its own.
func GateUploads(service *hub.Service, verifier identity.Verifier, slots int, wait time.Duration, next http.Handler) http.Handler {
	h := newHandler(service, verifier, identity.ServiceVerifiers{}, nil)
	h.uploads = make(chan struct{}, slots)
	h.uploadWait = wait
	return h.gateUploads(next)
}

// AttachmentError is the mapping of attachment failures to Connect errors.
func AttachmentError(err error) error { return attachmentError(err) }
