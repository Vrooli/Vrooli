Param(
    [string]$ModulePath = "scenarios/meta-optimization-manager/cli",
    [string]$Name = "meta-optimization-manager",
    [string]$Manifest = "scenarios/meta-optimization-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
