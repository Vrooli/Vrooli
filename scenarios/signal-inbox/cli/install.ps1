Param(
    [string]$ModulePath = "scenarios/signal-inbox/cli",
    [string]$Name = "signal-inbox",
    [string]$Manifest = "scenarios/signal-inbox/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
