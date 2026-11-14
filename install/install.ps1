param(
	[string]$Version = $env:VROOLI_VERSION,
	[string]$InstallDir = $env:VROOLI_INSTALL_DIR,
	[string]$ReleaseBaseUrl = $env:VROOLI_RELEASE_BASE_URL,
	[switch]$RunSetup
)

$ErrorActionPreference = 'Stop'
$releasePublicModulus = 'zTHpnCxVTVn7wSr/ZCplJ5U07JXI5BLwpXoz2HVljpD4Rxv3A++PQ3E6Yje0C/F7AKH6+KAToETwPxiFxOQZVuU9ZYJXDMOIxEvOqu0yBKaQC2o2sv+JTmoWCCmxf4ADk2JQ/zMvHZpy9tO0Z1Tg1dGORGV9UaKd7l5Lm7Aam0VtJ6NDXuzwbTb+Ag+98xiM88MJX1RcJVdM5auwExKgqmHsBiFIL48jRCaJYMDtk5uB7UkFw9QsqbG4PM2J7lGnf5UgyIcM7KXH5HU009rqapqFy9dt6BlAS4xByEpOB0uf7v79zdNqRiKPhC0Zu3rT4mfslhz9TQj2q1C/zfOHXflKh/DY9qvhlD6rXbRCrIf31PP1ZVzu4sryQxxrPEAfNsdns4mmmSSGwVkKIn+/dOCvbCEAy83Mxt5XN66Yn94rKO/8Z2V3qer7789Fq39fJC4fkbngFkLqpG9HxSl5xOWW4L6Cfs7uecjvPUcxOqWkcpFR3PTM9gcGdqonbagL'

function Test-EmbeddedManifestSignature([string]$Manifest, [string]$SignatureBase64) {
	$rsa = New-Object System.Security.Cryptography.RSACryptoServiceProvider
	try {
		$params = New-Object System.Security.Cryptography.RSAParameters
		$params.Modulus = [Convert]::FromBase64String($releasePublicModulus)
		$params.Exponent = [byte[]](1, 0, 1)
		$rsa.ImportParameters($params)
		$data = [IO.File]::ReadAllBytes($Manifest)
		$signature = [Convert]::FromBase64String((Get-Content -Raw -LiteralPath $SignatureBase64).Trim())
		if (-not $rsa.VerifyData($data, 'SHA256', $signature)) { throw 'Release signature verification failed.' }
	}
	finally { $rsa.Dispose() }
}

function Get-EmbeddedExpectedChecksum([string]$Manifest, [string]$Asset) {
	foreach ($line in Get-Content -LiteralPath $Manifest) {
		$parts = $line -split '\s+', 2
		if ($parts.Count -eq 2 -and $parts[1].TrimStart('*') -eq $Asset) { return $parts[0] }
	}
	throw "Release checksum manifest does not contain $Asset"
}
$repository = if ($env:VROOLI_GITHUB_REPOSITORY) { $env:VROOLI_GITHUB_REPOSITORY } else { 'Vrooli/Vrooli' }
if (-not $Version) { $Version = 'latest' }
if (-not $ReleaseBaseUrl) {
	if ($Version -eq 'latest') { $ReleaseBaseUrl = "https://github.com/$repository/releases/latest/download" }
	else {
		$tag = if ($Version.StartsWith('v')) { $Version } else { "v$Version" }
		$ReleaseBaseUrl = "https://github.com/$repository/releases/download/$tag"
	}
}
$ReleaseBaseUrl = $ReleaseBaseUrl.TrimEnd('/')
$workDir = Join-Path ([IO.Path]::GetTempPath()) ("vrooli-install-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $workDir | Out-Null
try {
	$helperAsset = 'vrooli-install-lib.ps1'
	$manifestAsset = 'SHA256SUMS'
	$signatureAsset = 'SHA256SUMS.sig'
	$helper = Join-Path $workDir $helperAsset
	$manifest = Join-Path $workDir $manifestAsset
	$signature = Join-Path $workDir $signatureAsset
	Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseBaseUrl/$helperAsset" -OutFile $helper
	Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseBaseUrl/$manifestAsset" -OutFile $manifest
	Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseBaseUrl/$signatureAsset" -OutFile $signature

	# Authenticate the helper before dot-sourcing it. Downloaded PowerShell is
	# code, so executing it before verification would defeat release signing.
	Test-EmbeddedManifestSignature $manifest $signature
	$expectedHelper = Get-EmbeddedExpectedChecksum $manifest $helperAsset
	$actualHelper = (Get-FileHash -Algorithm SHA256 -LiteralPath $helper).Hash.ToLowerInvariant()
	if ($actualHelper -ne $expectedHelper.ToLowerInvariant()) { throw 'Installer helper checksum mismatch.' }
	. $helper
	Test-VrooliManifestSignature $manifest $signature

	$arch = Get-VrooliArchitecture
	$asset = "vrooli_windows_$arch.exe"
	$sidecarAsset = "$asset.fp"
	$binary = Join-Path $workDir $asset
	$sidecar = Join-Path $workDir $sidecarAsset
	Invoke-VrooliDownload "$ReleaseBaseUrl/$asset" $binary
	Invoke-VrooliDownload "$ReleaseBaseUrl/$sidecarAsset" $sidecar
	Test-VrooliChecksum $manifest $asset $binary
	Test-VrooliChecksum $manifest $sidecarAsset $sidecar

	# Managed-service servers are separately signed release assets. Keep them in
	# a versioned per-user store so a native control plane never needs a source
	# checkout or an unsigned host-installed server binary.
	$resourceIndexAsset = 'resource-artifacts-v1.txt'
	$indexManifestLine = Get-Content -LiteralPath $manifest | Where-Object { ($_ -split '\s+', 2)[1].TrimStart('*') -eq $resourceIndexAsset } | Select-Object -First 1
	if ($indexManifestLine) {
		$resourceIndex = Join-Path $workDir $resourceIndexAsset
		Invoke-VrooliDownload "$ReleaseBaseUrl/$resourceIndexAsset" $resourceIndex
		Test-VrooliChecksum $manifest $resourceIndexAsset $resourceIndex
		$artifactRoot = if ($env:VROOLI_RESOURCE_ARTIFACT_DIR) { $env:VROOLI_RESOURCE_ARTIFACT_DIR } else { Join-Path $HOME '.vrooli\artifacts' }
		foreach ($line in Get-Content -LiteralPath $resourceIndex) {
			if ([string]::IsNullOrWhiteSpace($line)) { continue }
			$fields = $line -split "`t", 5
			if ($fields.Count -ne 5 -or ($fields | Where-Object { $_ -notmatch '^[A-Za-z0-9._-]+$' -or $_.Contains('..') })) {
				throw 'Resource artifact index contains unsafe fields.'
			}
			$resourceName, $resourceVersion, $resourceOS, $resourceArch, $resourceAsset = $fields
			if ($resourceOS -ne 'windows' -or $resourceArch -ne $arch) { continue }
			$serverPath = Join-Path $workDir $resourceAsset
			Invoke-VrooliDownload "$ReleaseBaseUrl/$resourceAsset" $serverPath
			Test-VrooliChecksum $manifest $resourceAsset $serverPath
			$destinationDir = Join-Path (Join-Path (Join-Path $artifactRoot $resourceName) $resourceVersion) ''
			New-Item -ItemType Directory -Force -Path $destinationDir | Out-Null
			Move-Item -Force -LiteralPath $serverPath -Destination (Join-Path $destinationDir $resourceAsset)
			Write-Output "Installed authenticated $resourceName service artifact to $(Join-Path $destinationDir $resourceAsset)"
		}
	}

	if (-not $InstallDir) { $InstallDir = Get-VrooliDefaultInstallDir }
	New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
	Move-Item -Force -LiteralPath $sidecar -Destination (Join-Path $InstallDir 'vrooli.exe.fp')
	Move-Item -Force -LiteralPath $binary -Destination (Join-Path $InstallDir 'vrooli.exe')
	Write-Output "Installed authenticated Vrooli CLI to $(Join-Path $InstallDir 'vrooli.exe')"
	Write-Output "Add $InstallDir to PATH. Full project setup is supported through the POSIX installer in WSL2, not native Windows."
	if ($RunSetup) { throw 'Native Windows vrooli setup is not supported; run the POSIX installer inside WSL2.' }
}
finally {
	Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $workDir
}
