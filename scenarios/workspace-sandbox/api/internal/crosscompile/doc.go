// Package crosscompile hosts the workspace-sandbox OS-seam cross-compile
// gate. It carries no runtime code; crosscompile_test.go builds the whole
// api module for the non-Linux targets the scenario must stay portable to
// (darwin/arm64 and windows/amd64), so a Linux-only syscall that leaks
// outside a //go:build-tagged file fails the unit suite instead of only
// surfacing on a developer's Mac or a Windows node. Mirror of the
// `make cross-compile` target in the scenario Makefile.
package crosscompile
