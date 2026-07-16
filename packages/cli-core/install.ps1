param(
	[Parameter(Mandatory = $true, Position = 0)]
	[string]$ModulePath,
	[string]$Name,
	[string]$Manifest,
	[string]$InstallDir,
	[string]$AppRoot
)

. (Join-Path $PSScriptRoot 'install/Platform.ps1')

$repoRoot = $AppRoot
if (-not $repoRoot) {
	$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "../..")).Path
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
	Write-Error "Go toolchain is required to build the CLI."
	exit 1
}

if (-not $InstallDir) {
	$InstallDir = Get-VrooliDefaultInstallDir
}

if (-not [System.IO.Path]::IsPathRooted($ModulePath)) {
	$ModulePath = Join-Path $repoRoot $ModulePath
}

if (-not (Test-Path (Join-Path $ModulePath "go.mod"))) {
	Write-Error "Module path must contain go.mod: $ModulePath"
	exit 1
}

$manifestPath = ""
if ($Manifest) {
	if (-not [System.IO.Path]::IsPathRooted($Manifest)) {
		$manifestPath = Join-Path $repoRoot $Manifest
	}
	else {
		$manifestPath = $Manifest
	}
	if (-not (Test-Path $manifestPath -PathType Leaf)) {
		Write-Error "Manifest path must contain a file: $manifestPath"
		exit 1
	}
}

if (-not $Name -or $Name -eq "") {
	$base = Split-Path $ModulePath -Leaf
	$parent = Split-Path (Split-Path $ModulePath -Parent) -Leaf
	if ($base -eq "cli") {
		$Name = $parent
	} else {
		$Name = $base
	}
}

$installerDir = Join-Path $repoRoot "packages/cli-core"
$installerTarget = "./cmd/cli-installer"
if ($env:CLI_CORE_VERSION) {
	$installerTarget = "github.com/vrooli/cli-core/cmd/cli-installer@$($env:CLI_CORE_VERSION)"
	$installerDir = $repoRoot
}

Write-Output "Building $Name from $ModulePath..."

Push-Location $installerDir
try {
	$args = @("run", $installerTarget, "--module", $ModulePath, "--name", $Name, "--install-dir", $InstallDir)
	if ($manifestPath) {
		$args += @("--manifest", $manifestPath)
	}
	& go @args
}
finally {
	Pop-Location
}
