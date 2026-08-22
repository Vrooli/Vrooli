package manifest

// EffectiveAcceleration returns the resource's accelerator declaration. nil
// means the resource declares no accelerator and holds no capacity claim.
//
// It exists as a named accessor rather than a bare field read because it is the
// one question every consumer asks, and because that question used to have
// three answers in three places.
func (m ResourceManifest) EffectiveAcceleration() *AccelerationSpec { return m.Acceleration }
