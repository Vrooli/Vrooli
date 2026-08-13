Param(
    [string]$ModulePath = "scenarios/money-ledger/cli",
    [string]$Name = "money-ledger",
    [string]$Manifest = "scenarios/money-ledger/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
