Param(
    [string]$ModulePath = "resources/gemini/cli",
    [string]$Name = "resource-gemini"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
