Param(
    [string]$ModulePath = "scenarios/architecture-cartographer/cli",
    [string]$Name = "architecture-cartographer",
    [string]$Manifest = "scenarios/architecture-cartographer/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
