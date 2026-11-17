package cliinstall

import "errors"

// These exported entrypoints keep command/API wiring separate from the
// hashed ordinary bundle transaction owner.
func InstallPreparedBundle(sourceDir, binDir, manifestSHA string) error {
	return installPreparedBundle(sourceDir, binDir, manifestSHA)
}

func InstallPreparedBundleWithSource(sourceDir, binDir, manifestSHA, sourceRoot string) error {
	return installPreparedBundleWithSource(sourceDir, binDir, manifestSHA, sourceRoot)
}

func InstallPreparedBundleSameContract(sourceDir, binDir, manifestSHA string) error {
	return installPreparedBundleSameContract(sourceDir, binDir, manifestSHA)
}

func InstallPreparedBundleSameContractWithSource(sourceDir, binDir, manifestSHA, sourceRoot string) error {
	return installPreparedBundleSameContractWithSource(sourceDir, binDir, manifestSHA, sourceRoot)
}

func InstallPreparedBundleSameContractWithSourceChecked(sourceDir, binDir, manifestSHA, sourceRoot string, check func() error) error {
	return installPreparedBundleSameContractWithSourceChecked(sourceDir, binDir, manifestSHA, sourceRoot, check)
}

func ResumePreparedBundle(binDir string) error {
	return resumePreparedBundle(binDir)
}

func AbortPreparedBundle(binDir string) error {
	return abortPreparedBundle(binDir)
}

func ordinaryBundleNoRecoveryError() error {
	return errors.New("no ordinary bundle install needs recovery")
}
