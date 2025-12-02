param(
	[Parameter(Mandatory = $true, Position = 0)]
	[string]$ModulePath,
	[string]$Name,
	[string]$InstallDir
)

$appRoot = $env:APP_ROOT
if (-not $appRoot) {
	$appRoot = (Resolve-Path (Join-Path $PSScriptRoot "../.."))
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
	Write-Error "Go toolchain is required to build the CLI."
	exit 1
}

if (-not $InstallDir) {
	$home = $env:HOME
	if (-not $home) { $home = $env:USERPROFILE }
	if (-not $home) { $home = [Environment]::GetFolderPath('UserProfile') }
	if (-not $home) {
		Write-Error "Unable to resolve home directory for install path."
		exit 1
	}
	$InstallDir = Join-Path $home ".vrooli/bin"
}

if (-not [System.IO.Path]::IsPathRooted($ModulePath)) {
	$ModulePath = Join-Path $appRoot $ModulePath
}

if (-not (Test-Path (Join-Path $ModulePath "go.mod"))) {
	Write-Error "Module path must contain go.mod: $ModulePath"
	exit 1
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

$installerDir = Join-Path $appRoot "packages/cli-core"
$installerTarget = "./cmd/cli-installer"
if ($env:CLI_CORE_VERSION) {
	$installerTarget = "github.com/vrooli/cli-core/cmd/cli-installer@$($env:CLI_CORE_VERSION)"
	$installerDir = $appRoot
}

Write-Output "Building $Name from $ModulePath..."

Push-Location $installerDir
try {
	& go run $installerTarget --module $ModulePath --name $Name --install-dir $InstallDir
}
finally {
	Pop-Location
}

