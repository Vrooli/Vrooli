param(
	[string]$InstallDir
)

$appRoot = $env:APP_ROOT
if (-not $appRoot) {
	$appRoot = (Resolve-Path (Join-Path $PSScriptRoot "../../.."))
}

$installer = Join-Path $appRoot "packages/cli-core/install.ps1"
$parameters = @{
	ModulePath = "scenarios/scenario-completeness-scoring/cli"
	Name       = "scenario-completeness-scoring"
}
if ($InstallDir) {
	$parameters.InstallDir = $InstallDir
}

& $installer @parameters
