Param(
    [string]$ModulePath = "scenarios/scenario-to-mcp/cli",
    [string]$Name = "scenario-to-mcp",
    [string]$Manifest = "scenarios/scenario-to-mcp/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
