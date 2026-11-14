//go:build linux

package hostwireless

import (
	"context"
	"encoding/json"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func radioBlueZCall(ctx context.Context, t *durableWireless, service, object, iface, method string, args ...string) ([]byte, error) {
	command := []string{"--system", "--no-pager", "--timeout=5", "--auto-start=no", "--allow-interactive-authorization=no", "--json=short", "call", service, object, iface, method}
	return t.radioSystem().command(ctx, "/usr/bin/busctl", append(command, args...)...)
}

func observeBlueZPeer(ctx context.Context, t *durableWireless, r *WirelessRadioObservation) {
	if len(r.Controllers) == 0 {
		return
	}
	// NameHasOwner talks only to the existing system bus. The following
	// explicit call also has NO_AUTO_START, closing the owner-exit race.
	out, err := radioBlueZCall(ctx, t, "org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus", "NameHasOwner", "s", "org.bluez")
	if err != nil {
		return
	}
	var owner struct {
		Type string `json:"type"`
		Data []bool `json:"data"`
	}
	if err = json.Unmarshal(out, &owner); err != nil || owner.Type != "b" || len(owner.Data) != 1 || !owner.Data[0] {
		return
	}
	out, err = radioBlueZCall(ctx, t, "org.bluez", "/", "org.freedesktop.DBus.ObjectManager", "GetManagedObjects")
	if err != nil {
		return
	}
	verified, err := blueZPeerEvidence(out, r.Controllers)
	if err != nil {
		r.Details = append(r.Details, "Passive BlueZ peer evidence unavailable: "+err.Error())
		return
	}
	if verified {
		r.FunctionVerified = true
		r.Details = append(r.Details, "Selected powered Bluetooth adapter reports a paired and currently connected peer; discovery/resume are separate proofs")
	}
}

func blueZPeerEvidence(data []byte, controllers []string) (bool, error) {
	observed, err := hostinventory.ParseLinuxBluetoothControllers(data)
	if err != nil {
		return false, err
	}
	for _, name := range controllers {
		controller := observed[name]
		if controller.PowerKnown && controller.Powered && controller.PeerConnected {
			return true, nil
		}
	}
	return false, nil
}
