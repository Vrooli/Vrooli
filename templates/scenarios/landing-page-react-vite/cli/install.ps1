Param(
    [string]$ModulePath = "",
    [string]$Name = "",
    [string]$Manifest = "scenarios/{{SCENARIO_ID}}/.vrooli/service.json"
)

if (-not $ModulePath -or $ModulePath -eq "") {
    $ModulePath = (Resolve-Path $PSScriptRoot | Select-Object -First 1).Path
}

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name -Manifest $Manifest
