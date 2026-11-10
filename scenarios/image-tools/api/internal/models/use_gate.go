package models

import "github.com/vrooli/vrooli/packages/capacity"

// ErrModelInUse is the shared, cross-process artifact-use refusal.
var ErrModelInUse = capacity.ErrAssetInUse

// ModelUseGate protects installed weights across inference and mutation in
// every process sharing the collection. The capacity client owns the lease.
type ModelUseGate = capacity.UseGate

func NewModelUseGate(root string) *ModelUseGate { return &capacity.UseGate{Root: root} }
