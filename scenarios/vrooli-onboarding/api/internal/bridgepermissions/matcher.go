package bridgepermissions

import "strings"

func (*Provider) MatchesPermissionID(id string) bool {
	return strings.HasPrefix(id, "bridge/node-scopes/")
}
