param(
	[string]$InstallDir
)

$appRoot = $env:APP_ROOT
if (-not $appRoot) {
	$appRoot = (Resolve-Path (Join-Path $PSScriptRoot "../../.."))
}

$installer = Join-Path $appRoot "packages/cli-core/install.ps1"
$parameters = @{
	ModulePath = "scenarios/test-genie/cli"
	Name       = "test-genie"
}
if ($InstallDir) {
	$parameters.InstallDir = $InstallDir
}

& $installer @parameters
