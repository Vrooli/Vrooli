Param(
    [string]$ModulePath = "resources/mail-in-a-box/cli",
    [string]$Name = "resource-mail-in-a-box",
    [string]$Manifest = "resources/mail-in-a-box/resource.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
