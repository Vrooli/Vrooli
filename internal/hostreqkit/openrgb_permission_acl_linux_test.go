//go:build linux

package hostreqkit

import (
	"context"
	"encoding/binary"
	"testing"
	"time"
)

// Linux access-ACL xattr wire data: owner rw, named invoking user rw,
// owning group none, mask rw, other none. The mask yields mode0660 without
// granting the owning group any access.
func permissionReadbackACL(mask uint16) []byte {
	data := make([]byte, 4+5*8)
	binary.LittleEndian.PutUint32(data, 2)
	entries := [][3]uint32{{1, 6, 0xffffffff}, {2, 6, 1000}, {4, 0, 0xffffffff}, {16, uint32(mask), 0xffffffff}, {32, 0, 0xffffffff}}
	for i, e := range entries {
		off := 4 + i*8
		binary.LittleEndian.PutUint16(data[off:], uint16(e[0]))
		binary.LittleEndian.PutUint16(data[off+2:], uint16(e[1]))
		binary.LittleEndian.PutUint32(data[off+4:], e[2])
	}
	return data
}

func TestOpenRGBPermissionRetryAcceptsPreservedOwnerACLMask(t *testing.T) {
	f := newPermissionFixture(t)
	for d, s := range f.snapshots {
		s.ACL = permissionReadbackACL(6)
		s.Mode = 0o660
		f.snapshots[d] = s
	}
	p := f.plan(t)
	f.failTrigger = true
	if got, err := f.e.apply(context.Background(), p); err == nil || !got.Pending {
		t.Fatalf("fixture did not retain pending sameplan: %+v %v", got, err)
	}
	f.failTrigger = false
	run := f.e.run
	f.e.run = func(ctx context.Context, args ...string) error {
		if err := run(ctx, args...); err != nil {
			return err
		}
		if args[0] == "trigger" {
			for _, n := range f.nodes {
				if n.SysPath == args[len(args)-1] {
					s := f.snapshots[n.Selector]
					s.Mode = 0o660
					s.ACL = permissionReadbackACL(6)
					f.snapshots[n.Selector] = s
				}
			}
		}
		return nil
	}
	// An already retained transaction remains retryable after plan lifetime.
	f.now = f.now.Add(30 * time.Minute)
	result, err := f.e.apply(context.Background(), p)
	if err != nil || !result.Applied || result.Pending || result.PlanID != p.ID {
		t.Fatalf("preserved owner ACL retry: %+v %v", result, err)
	}
}

func accessACLFixture(entries ...[3]uint32) []byte {
	data := make([]byte, 4+8*len(entries))
	binary.LittleEndian.PutUint32(data, 2)
	for i, e := range entries {
		off := 4 + 8*i
		binary.LittleEndian.PutUint16(data[off:], uint16(e[0]))
		binary.LittleEndian.PutUint16(data[off+2:], uint16(e[1]))
		binary.LittleEndian.PutUint32(data[off+4:], e[2])
	}
	return data
}

func TestOpenRGBPermissionACLRejectsNewOrUnknownAccess(t *testing.T) {
	undefined := uint32(0xffffffff)
	baseline := openRGBNodeSnapshot{UID: 0, GID: 51, Mode: 0o660, ACL: permissionReadbackACL(6)}
	granted := baseline
	granted.UID = 1000
	if err := verifyOpenRGBGrantedNode(granted, baseline, 1000); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*openRGBNodeSnapshot, *openRGBNodeSnapshot){
		"unmasked-owning-group": func(now, prior *openRGBNodeSnapshot) {
			now.ACL = accessACLFixture([3]uint32{1, 6, undefined}, [3]uint32{2, 6, 1000}, [3]uint32{4, 6, undefined}, [3]uint32{16, 6, undefined}, [3]uint32{32, 0, undefined})
		},
		"mode-is-real-group-grant": func(now, prior *openRGBNodeSnapshot) { now.ACL = nil },
		"foreign-user-added": func(now, prior *openRGBNodeSnapshot) {
			now.ACL = accessACLFixture([3]uint32{1, 6, undefined}, [3]uint32{2, 6, 1000}, [3]uint32{2, 6, 2000}, [3]uint32{4, 0, undefined}, [3]uint32{16, 6, undefined}, [3]uint32{32, 0, undefined})
		},
		"foreign-user-unmasked": func(now, prior *openRGBNodeSnapshot) {
			prior.Mode = 0o600
			prior.ACL = accessACLFixture([3]uint32{1, 6, undefined}, [3]uint32{2, 4, 2000}, [3]uint32{4, 0, undefined}, [3]uint32{16, 0, undefined}, [3]uint32{32, 0, undefined})
			now.ACL = accessACLFixture([3]uint32{1, 6, undefined}, [3]uint32{2, 4, 2000}, [3]uint32{4, 0, undefined}, [3]uint32{16, 6, undefined}, [3]uint32{32, 0, undefined})
		},
		"restrictive-user-removed": func(now, prior *openRGBNodeSnapshot) {
			prior.ACL = accessACLFixture([3]uint32{1, 6, undefined}, [3]uint32{2, 0, 2000}, [3]uint32{4, 6, undefined}, [3]uint32{16, 6, undefined}, [3]uint32{32, 0, undefined})
			now.ACL = accessACLFixture([3]uint32{1, 6, undefined}, [3]uint32{4, 6, undefined}, [3]uint32{16, 6, undefined}, [3]uint32{32, 0, undefined})
		},
		"new-other": func(now, prior *openRGBNodeSnapshot) {
			now.Mode = 0o662
			now.ACL = accessACLFixture([3]uint32{1, 6, undefined}, [3]uint32{2, 6, 1000}, [3]uint32{4, 0, undefined}, [3]uint32{16, 6, undefined}, [3]uint32{32, 2, undefined})
		},
		"group-owner-changed": func(now, prior *openRGBNodeSnapshot) { now.GID++ },
		"wrong-owner":         func(now, prior *openRGBNodeSnapshot) { now.UID++ },
		"owner-not-writable":  func(now, prior *openRGBNodeSnapshot) { now.Mode = 0o460 },
		"special-mode":        func(now, prior *openRGBNodeSnapshot) { now.Mode |= 0o4000 },
		"executable-mode":     func(now, prior *openRGBNodeSnapshot) { now.Mode |= 0o10 },
		"malformed-prior":     func(now, prior *openRGBNodeSnapshot) { prior.ACL = []byte("unknown ACL") },
		"malformed-current":   func(now, prior *openRGBNodeSnapshot) { now.ACL = []byte("unknown ACL") },
		"mode-mask-disagree":  func(now, prior *openRGBNodeSnapshot) { now.Mode = 0o600 },
		"missing-mask": func(now, prior *openRGBNodeSnapshot) {
			now.ACL = accessACLFixture([3]uint32{1, 6, undefined}, [3]uint32{2, 6, 1000}, [3]uint32{4, 6, undefined}, [3]uint32{32, 0, undefined})
		},
		"duplicate-user": func(now, prior *openRGBNodeSnapshot) {
			now.ACL = accessACLFixture([3]uint32{1, 6, undefined}, [3]uint32{2, 6, 1000}, [3]uint32{2, 6, 1000}, [3]uint32{4, 0, undefined}, [3]uint32{16, 6, undefined}, [3]uint32{32, 0, undefined})
		},
		"unknown-tag": func(now, prior *openRGBNodeSnapshot) {
			now.ACL = append([]byte(nil), now.ACL...)
			binary.LittleEndian.PutUint16(now.ACL[12:], 64)
		},
		"unknown-permission": func(now, prior *openRGBNodeSnapshot) {
			now.ACL = append([]byte(nil), now.ACL...)
			binary.LittleEndian.PutUint16(now.ACL[14:], 8)
		},
		"invalid-object-id": func(now, prior *openRGBNodeSnapshot) {
			now.ACL = append([]byte(nil), now.ACL...)
			binary.LittleEndian.PutUint32(now.ACL[8:], 0)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			now, prior := granted, baseline
			change(&now, &prior)
			if err := verifyOpenRGBGrantedNode(now, prior, 1000); err == nil {
				t.Fatal("new or unknown permission state was acknowledged")
			}
		})
	}
}

func TestOpenRGBPermissionACLRetainsExistingNamedGrants(t *testing.T) {
	undefined := uint32(0xffffffff)
	data := accessACLFixture([3]uint32{1, 6, undefined}, [3]uint32{2, 4, 2000}, [3]uint32{4, 0, undefined}, [3]uint32{8, 6, 3000}, [3]uint32{16, 6, undefined}, [3]uint32{32, 0, undefined})
	prior := openRGBNodeSnapshot{UID: 0, GID: 51, Mode: 0o660, ACL: data}
	now := prior
	now.UID = 1000
	if err := verifyOpenRGBGrantedNode(now, prior, 1000); err != nil {
		t.Fatal(err)
	}
	now.ACL = append([]byte(nil), data...)
	binary.LittleEndian.PutUint16(now.ACL[14:], 0)
	if err := verifyOpenRGBGrantedNode(now, prior, 1000); err == nil {
		t.Fatal("lost ambient named-user grant was acknowledged")
	}
}
