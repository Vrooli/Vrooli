Param(
    [string]$ModulePath = "scenarios/program-runtime/cli",
    [string]$Name = "program-runtime",
    [string]$Manifest = "scenarios/program-runtime/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
