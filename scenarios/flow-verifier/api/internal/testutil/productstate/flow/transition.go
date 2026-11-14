package flow

import "flow-verifier/internal/testutil/productstate/flow/generated"

func TransitionProduct(status generated.ProductStatus, event generated.ProductEvent) (generated.ProductStatus, error) {
	return generated.TransitionProductStatus(status, event)
}
