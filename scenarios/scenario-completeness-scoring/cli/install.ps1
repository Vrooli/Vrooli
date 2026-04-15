param(
	[string]$InstallDir
)

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "../../..")).Path

$installer = Join-Path $repoRoot "packages/cli-core/install.ps1"
$parameters = @{
	ModulePath = "scenarios/scenario-completeness-scoring/cli"
	Name       = "scenario-completeness-scoring"
}
if ($InstallDir) {
	$parameters.InstallDir = $InstallDir
}

& $installer @parameters
