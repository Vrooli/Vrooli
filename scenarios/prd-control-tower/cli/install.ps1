param(
	[string]$InstallDir
)

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "../../..")).Path

$modulePath = "scenarios/prd-control-tower/cli"

Push-Location $repoRoot
try {
	$script = Join-Path $repoRoot "packages/cli-core/install.ps1"
	if (-not (Test-Path $script)) {
		Write-Error "cli-core installer not found at $script"
		exit 1
	}

	if ($InstallDir) {
		& $script $modulePath -Name "prd-control-tower" -InstallDir $InstallDir
	} else {
		& $script $modulePath -Name "prd-control-tower"
	}
}
finally {
	Pop-Location
}
