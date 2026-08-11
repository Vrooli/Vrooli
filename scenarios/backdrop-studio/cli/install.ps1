Param(
    [string]$ModulePath = "scenarios/backdrop-studio/cli",
    [string]$Name = "backdrop-studio",
    [string]$Manifest = "scenarios/backdrop-studio/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
