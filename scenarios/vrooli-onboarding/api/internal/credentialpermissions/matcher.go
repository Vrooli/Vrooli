package credentialpermissions

import "strings"

func (*Provider) MatchesPermissionID(id string) bool {
	return strings.HasPrefix(id, "bridge/credential-grant/") || strings.HasPrefix(id, ActionPrefix)
}
