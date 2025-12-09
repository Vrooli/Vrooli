Param([String[]]$Rest)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Resolve-Path (Join-Path $ScriptDir "..\..")
$CliCore = Join-Path $RepoRoot "packages\cli-core\install.ps1"

if (-not (Test-Path $CliCore)) {
    Write-Error "cli-core installer not found at $CliCore"
}

& $CliCore -ModulePath $ScriptDir -Name "resource-sqlite" @Rest
