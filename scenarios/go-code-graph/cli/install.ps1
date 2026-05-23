Param(
    [string]$ModulePath = "scenarios/go-code-graph/cli",
    [string]$Name = "go-code-graph",
    [string]$Manifest = "scenarios/go-code-graph/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
