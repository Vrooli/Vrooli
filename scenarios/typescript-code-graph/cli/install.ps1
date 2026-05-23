Param(
    [string]$ModulePath = "scenarios/typescript-code-graph/cli",
    [string]$Name = "typescript-code-graph",
    [string]$Manifest = "scenarios/typescript-code-graph/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
