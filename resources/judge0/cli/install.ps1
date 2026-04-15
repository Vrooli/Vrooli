Param(
    [string]$ModulePath = "resources/judge0/cli",
    [string]$Name = "resource-judge0"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
