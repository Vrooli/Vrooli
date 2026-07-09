Param(
    [string]$ModulePath = "scenarios/template-manager/cli",
    [string]$Name = "template-manager",
    [string]$Manifest = "scenarios/template-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
