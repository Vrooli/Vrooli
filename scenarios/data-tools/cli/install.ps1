Param(
    [string]$ModulePath = "scenarios/data-tools/cli",
    [string]$Name = "data-tools",
    [string]$Manifest = "scenarios/data-tools/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
