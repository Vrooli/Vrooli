package shapes

import (
	shapesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shapes/shapes_v1connect"
	"program-runtime/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "shapes_list", Path: shapesconnect.ShapeServiceListShapesProcedure, Method: "POST", Summary: "List observed and nominated binding shapes.", Category: "shapes"},
	{ID: "shapes_get", Path: shapesconnect.ShapeServiceGetShapeProcedure, Method: "POST", Summary: "Read one binding shape.", Category: "shapes"},
	{ID: "shapes_expire", Path: shapesconnect.ShapeServiceExpireShapesProcedure, Method: "POST", Summary: "Expire observed binding shapes.", Category: "shapes"},
}
