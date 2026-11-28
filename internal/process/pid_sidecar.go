package process

import "strconv"

// The sidecar remains readable by legacy process consumers across activation.
// Its source is pinned along with the process-records persistent contract.
func pidSidecarName(step string) string { return step + ".pid" }
func encodePIDSidecar(pid int) []byte   { return []byte(strconv.Itoa(pid) + "\n") }
