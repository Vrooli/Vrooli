Param(
    [string]$ModulePath = "scenarios/funnel-builder/cli",
    [string]$Name = "funnel-builder",
    [string]$Manifest = "scenarios/funnel-builder/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
