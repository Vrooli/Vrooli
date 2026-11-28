package control

// closeState is the close protocol shared by every owner-held resource that
// needs an external, fallible close (a pairing handshake, a native recording):
//
//	open ──beginClose──▶ closing ──closed──▶ (removed)
//	                        │
//	                        └─closeFailed──▶ uncertain
//
// A resource whose close failed is uncertain, never open again: the owner may
// have consumed the native handle, so the close is never replayed
// automatically and the resource stays visible as a lifecycle blocker until an
// owner-specific recovery resolves it.
type closeState uint8

const (
	stateOpen closeState = iota
	stateClosing
	stateUncertain
)

func (s closeState) String() string {
	switch s {
	case stateOpen:
		return "open"
	case stateClosing:
		return "closing"
	case stateUncertain:
		return "uncertain"
	}
	return "invalid"
}

// beginClose moves an open resource to closing. Any other state refuses: a
// closing resource already has an owner of the close, and an uncertain one must
// not be closed again blindly.
func (s closeState) beginClose() (closeState, bool) {
	if s != stateOpen {
		return s, false
	}
	return stateClosing, true
}

// closeResult applies the outcome of a close that beginClose admitted. It
// reports removed=true when the resource is gone; otherwise the failed close
// leaves it uncertain.
func (s closeState) closeResult(closeErr error) (next closeState, removed bool) {
	if s != stateClosing {
		return s, false
	}
	if closeErr == nil {
		return stateClosing, true
	}
	return stateUncertain, false
}

// usable reports whether the resource may still serve its normal operation.
func (s closeState) usable() bool { return s == stateOpen }
