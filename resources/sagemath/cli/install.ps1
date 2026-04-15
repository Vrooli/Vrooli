Param(
    [string]$ModulePath = "resources/sagemath/cli",
    [string]$Name = "resource-sagemath"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
