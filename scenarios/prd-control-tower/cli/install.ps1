param(
	[string]$InstallDir
)

$appRoot = $env:APP_ROOT
if (-not $appRoot) {
	$appRoot = (Resolve-Path (Join-Path $PSScriptRoot "../../.."))
}

$modulePath = "scenarios/prd-control-tower/cli"

Push-Location $appRoot
try {
	$script = Join-Path $appRoot "packages/cli-core/install.ps1"
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

