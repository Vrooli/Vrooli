Param(
    [string]$ModulePath = "scenarios/time-tools/cli",
    [string]$Name = "time-tools",
    [string]$Manifest = "scenarios/time-tools/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
