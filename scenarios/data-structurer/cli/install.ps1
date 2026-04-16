Param(
    [string]$ModulePath = "scenarios/data-structurer/cli",
    [string]$Name = "data-structurer",
    [string]$Manifest = "scenarios/data-structurer/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
