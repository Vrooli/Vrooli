Param(
    [string]$ModulePath = "scenarios/tunnel-manager/cli",
    [string]$Name = "tunnel-manager"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
