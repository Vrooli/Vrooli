Param(
    [string]$ModulePath = "resources/speaker-verification/cli",
    [string]$Name = "resource-speaker-verification",
    [string]$Manifest = "resources/speaker-verification/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
