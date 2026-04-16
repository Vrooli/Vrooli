Param(
    [string]$ModulePath = "resources/{{RESOURCE_NAME}}/cli",
    [string]$Name = "{{RESOURCE_CLI_COMMAND}}"
)

$repoRoot = (Resolve-Path "$PSScriptRoot/../../.." | Select-Object -First 1).Path

& "$repoRoot/packages/cli-core/install.ps1" -ModulePath $ModulePath -Name $Name
