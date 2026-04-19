Param(
    [string]$ModulePath = "scenarios/command-center/cli",
    [string]$Name = "command-center",
    [string]$Manifest = "scenarios/command-center/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
