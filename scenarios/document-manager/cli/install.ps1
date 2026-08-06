Param(
    [string]$ModulePath = "scenarios/document-manager/cli",
    [string]$Name = "document-manager",
    [string]$Manifest = "scenarios/document-manager/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
