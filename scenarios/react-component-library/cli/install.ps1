Param(
    [string]$ModulePath = "scenarios/react-component-library/cli",
    [string]$Name = "react-component-library",
    [string]$Manifest = "scenarios/react-component-library/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
