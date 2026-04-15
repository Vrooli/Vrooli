Param(
    [string]$ModulePath = "resources/vault/cli",
    [string]$Name = "resource-vault"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
