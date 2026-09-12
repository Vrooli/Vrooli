// Package runtime contains shared native-Go runtime primitives for resources.
//
// It is the control-plane-owned runtime layer that active resources should
// converge toward over time. Packages under this tree are intended to keep
// storage, env rendering, health probing, log path policy, and common
// operation semantics out of per-resource ad hoc implementations.
package runtime
