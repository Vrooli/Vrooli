Param(
    [string]$ModulePath = "resources/twilio/cli",
    [string]$Name = "resource-twilio",
    [string]$Manifest = "resources/twilio/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
