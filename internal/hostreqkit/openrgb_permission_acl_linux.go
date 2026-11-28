//go:build linux

package hostreqkit

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Linux's access-ACL wire format and mask/mode semantics are defined in
// include/uapi/linux/posix_acl_xattr.h and fs/posix_acl.c. Group mode bits
// encode the mask when an extended ACL exists, not the owning group's grant.
type openRGBAccessACL struct {
	owner, group, other, mask uint16
	users, groups             map[uint32]uint16
	extended                  bool
}

func decodeOpenRGBAccessACL(s openRGBNodeSnapshot) (openRGBAccessACL, error) {
	a := openRGBAccessACL{owner: uint16(s.Mode>>6) & 7, group: uint16(s.Mode>>3) & 7, other: uint16(s.Mode) & 7, mask: 7, users: map[uint32]uint16{}, groups: map[uint32]uint16{}}
	if len(s.ACL) == 0 {
		return a, nil
	}
	if len(s.ACL) < 28 || len(s.ACL) > 64<<10 || (len(s.ACL)-4)%8 != 0 || binary.LittleEndian.Uint32(s.ACL) != 2 {
		return a, errors.New("invalid Linux access-ACL header or size")
	}
	state := 0
	var lastUser, lastGroup uint32
	var haveUser, haveGroup, haveMask bool
	for off := 4; off < len(s.ACL); off += 8 {
		tag := binary.LittleEndian.Uint16(s.ACL[off:])
		perm := binary.LittleEndian.Uint16(s.ACL[off+2:])
		id := binary.LittleEndian.Uint32(s.ACL[off+4:])
		if perm&^uint16(7) != 0 {
			return a, errors.New("unknown access-ACL permission bits")
		}
		switch tag {
		case 1:
			if state != 0 || id != 0xffffffff {
				return a, errors.New("invalid access-ACL owner entry")
			}
			a.owner = perm
			state = 1
		case 2:
			if state != 1 || id == 0xffffffff || haveUser && id <= lastUser {
				return a, errors.New("invalid or duplicate access-ACL user entry")
			}
			a.users[id] = perm
			lastUser = id
			haveUser = true
		case 4:
			if state != 1 || id != 0xffffffff {
				return a, errors.New("invalid access-ACL owning-group entry")
			}
			a.group = perm
			state = 2
		case 8:
			if state != 2 || id == 0xffffffff || haveGroup && id <= lastGroup {
				return a, errors.New("invalid or duplicate access-ACL group entry")
			}
			a.groups[id] = perm
			lastGroup = id
			haveGroup = true
		case 16:
			if state != 2 || id != 0xffffffff {
				return a, errors.New("invalid access-ACL mask entry")
			}
			a.mask = perm
			haveMask = true
			state = 3
		case 32:
			if state != 3 && (state != 2 || haveUser || haveGroup) || id != 0xffffffff {
				return a, errors.New("missing mask or invalid access-ACL other entry")
			}
			a.other = perm
			state = 4
		default:
			return a, errors.New("unknown access-ACL tag")
		}
	}
	if state != 4 {
		return a, errors.New("incomplete access ACL")
	}
	a.extended = true
	groupMode := a.group
	if haveMask {
		groupMode = a.mask
	}
	if uint32(a.owner)<<6|uint32(groupMode)<<3|uint32(a.other) != s.Mode&0o777 {
		return a, errors.New("access ACL differs from observed mode bits")
	}
	return a, nil
}

// The only new principal admitted by this transaction is the selected owner.
// ACL identities are retained so removing a restrictive named entry cannot
// silently expose that principal through group/other fallback permissions.
func verifyOpenRGBGrantedNode(current, prior openRGBNodeSnapshot, owner uint32) error {
	if current.Selector != prior.Selector || current.UID != owner || current.GID != prior.GID || current.Mode&0o700 != 0o600 || current.Mode&^uint32(0o777) != 0 || current.Mode&0o111 != 0 {
		return errors.New("selected node owner, group or non-executable read/write mode differs")
	}
	before, err := decodeOpenRGBAccessACL(prior)
	if err != nil {
		return fmt.Errorf("trusted prior ACL: %w", err)
	}
	after, err := decodeOpenRGBAccessACL(current)
	if err != nil {
		return fmt.Errorf("observed ACL: %w", err)
	}
	sameGrant := func(now, old uint16) bool {
		if before.extended {
			return now == old
		}
		return now&^old == 0
	}
	if !sameGrant(after.group&after.mask, before.group&before.mask) || !sameGrant(after.other, before.other) {
		return errors.New("owning-group or other access changed beyond the retained permissions")
	}
	for id, perm := range after.users {
		if id == owner {
			if perm&^uint16(6) != 0 {
				return errors.New("unexpected invoking-owner ACL permissions")
			}
			continue
		}
		old, ok := before.users[id]
		if !ok || perm&after.mask != old&before.mask {
			return errors.New("foreign named-user access differs from retained ACL")
		}
	}
	for id := range before.users {
		if id != owner {
			if _, ok := after.users[id]; !ok {
				return errors.New("restrictive named-user ACL entry was removed")
			}
		}
	}
	if len(after.groups) != len(before.groups) {
		return errors.New("named-group ACL identities changed")
	}
	for id, perm := range after.groups {
		old, ok := before.groups[id]
		if !ok || perm&after.mask != old&before.mask {
			return errors.New("named-group access differs from retained ACL")
		}
	}
	// Chown changes the old owner's precedence too. Root already bypasses DAC;
	// for another old owner, conservatively bound its possible fallback grants.
	if prior.UID != 0 && prior.UID != owner {
		fallback := after.other | after.group&after.mask
		if named, ok := after.users[prior.UID]; ok {
			fallback = named & after.mask
		} else {
			for _, perm := range after.groups {
				fallback |= perm & after.mask
			}
		}
		if fallback&^before.owner != 0 {
			return errors.New("ownership change expanded the prior owner's effective access")
		}
	}
	return nil
}
