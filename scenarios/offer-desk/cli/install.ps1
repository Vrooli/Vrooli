Param(
    [string]$ModulePath = "scenarios/offer-desk/cli",
    [string]$Name = "offer-desk",
    [string]$Manifest = "scenarios/offer-desk/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
