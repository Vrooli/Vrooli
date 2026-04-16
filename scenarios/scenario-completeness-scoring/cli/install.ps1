param(
	[string]$InstallDir
)

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "../../..")).Path

$installer = Join-Path $repoRoot "packages/cli-core/install.ps1"
$parameters = @{
	ModulePath = "scenarios/scenario-completeness-scoring/cli"
	Name       = "scenario-completeness-scoring"
	Manifest   = "scenarios/scenario-completeness-scoring/.vrooli/service.json"
}
if ($InstallDir) {
	$parameters.InstallDir = $InstallDir
}

& $installer @parameters
