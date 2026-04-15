Param(
    [string]$ModulePath = "resources/redis/cli",
    [string]$Name = "resource-redis"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
