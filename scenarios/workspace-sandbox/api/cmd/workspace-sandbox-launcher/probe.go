package main

import "os/exec"

func probePlainUserns() error {
	return exec.Command("unshare", "-U", "-m", "-r", "true").Run()
}

func probeAppArmorUserns() error {
	return exec.Command("aa-exec", "-p", "vrooli-workspace-sandbox", "--", "unshare", "-U", "-m", "-r", "true").Run()
}
