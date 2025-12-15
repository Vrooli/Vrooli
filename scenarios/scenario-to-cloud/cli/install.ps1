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
	Name       = "scenario-to-cloud-bin"
}
if ($InstallDir) {
	$parameters.InstallDir = $InstallDir
}

& $installer @parameters

if (-not $InstallDir) {
	$home = $env:HOME
	if (-not $home) { $home = $env:USERPROFILE }
	if (-not $home) { $home = [Environment]::GetFolderPath('UserProfile') }
	$InstallDir = Join-Path $home ".vrooli/bin"
}

$wrapperPath = Join-Path $InstallDir "scenario-to-cloud.cmd"
$binPath = Join-Path $InstallDir "scenario-to-cloud-bin.exe"
if (-not (Test-Path $binPath)) {
	$binPath = Join-Path $InstallDir "scenario-to-cloud-bin"
}

$cmd = @"
@echo off
set VROOLI_LIFECYCLE_MANAGED=true
\"%~dp0$(Split-Path $binPath -Leaf)\" %*
"@

Set-Content -Path $wrapperPath -Value $cmd -Encoding ASCII
