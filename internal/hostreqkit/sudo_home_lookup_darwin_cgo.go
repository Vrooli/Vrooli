//go:build darwin && cgo

package hostreqkit

import osuser "os/user"

func lookupDarwinInvokingUser(uid string) (*osuser.User, error) { return osuser.LookupId(uid) }
