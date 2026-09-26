package lifecycle

import "github.com/vrooli/vrooli/internal/peerrecord"

// PeerRecord is the cross-tier discovery projection published by lifecycle.
// Credentials remain owned by their existing authorities.
type PeerRecord = peerrecord.Record

func writePeerRecord(home string, record PeerRecord) error {
	return peerrecord.Write(home, record)
}

func removePeerRecord(home, name string) error {
	return peerrecord.Remove(home, name)
}
