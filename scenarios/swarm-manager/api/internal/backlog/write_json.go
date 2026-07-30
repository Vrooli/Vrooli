package backlog

import "swarm-manager/internal/jsonutil"

func writeJSONRedacted(path string, value any) error {
	return jsonutil.WriteFileRedacted(path, value)
}
