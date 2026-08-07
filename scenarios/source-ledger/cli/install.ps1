Param(
    [string]$ModulePath = "scenarios/source-ledger/cli",
    [string]$Name = "source-ledger",
    [string]$Manifest = "scenarios/source-ledger/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
