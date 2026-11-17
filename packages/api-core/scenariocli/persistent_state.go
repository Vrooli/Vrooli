package scenariocli

import "reflect"

// PersistentTypes exposes the stored sidecar shape to the root control-plane
// contract without coupling this shared module to the control plane.
func PersistentTypes() []reflect.Type { return []reflect.Type{reflect.TypeFor[installMetadata]()} }
