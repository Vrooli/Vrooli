Param(
    [string]$ModulePath = "scenarios/prompt-injection-arena/cli",
    [string]$Name = "prompt-injection-arena",
    [string]$Manifest = "scenarios/prompt-injection-arena/.vrooli/service.json"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
