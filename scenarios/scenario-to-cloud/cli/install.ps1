param(
	[string]$InstallDir
)

$appRoot = $env:APP_ROOT
if (-not $appRoot) {
	$appRoot = (Resolve-Path (Join-Path $PSScriptRoot "../../.."))
}

$installer = Join-Path $appRoot "packages/cli-core/install.ps1"
$parameters = @{
	ModulePath = "scenarios/scenario-to-cloud/cli"
	Name       = "scenario-to-cloud"
}
if ($InstallDir) {
	$parameters.InstallDir = $InstallDir
}

& $installer @parameters
