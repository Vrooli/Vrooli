package main

import "landing-page-business-suite-api/internal/experimentation"

// The API composition root owns the process-wide default; variant-space
// parsing and validation are owned by the experimentation domain.
type (
	VariantSpace            = experimentation.VariantSpace
	AxisDefinition          = experimentation.AxisDefinition
	AxisVariant             = experimentation.AxisVariant
	VariantSpaceConstraints = experimentation.VariantSpaceConstraints
)

var defaultVariantSpace = experimentation.DefaultVariantSpace()
