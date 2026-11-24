//go:build !audio_lab

package server

func ComposeForBuild(d CompositionDeps) *Server { return Compose(d) }
