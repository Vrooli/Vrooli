Param(
    [string]$ModulePath = "scenarios/reference-react-vite/cli",
    [string]$Name = "reference-react-vite"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name 
