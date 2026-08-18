function Get-VrooliDefaultInstallDir {
	$homeDir = $env:HOME
	if (-not $homeDir) { $homeDir = $env:USERPROFILE }
	if (-not $homeDir) { $homeDir = [Environment]::GetFolderPath('UserProfile') }
	if (-not $homeDir) { throw 'Unable to resolve home directory for the Vrooli install path.' }
	return (Join-Path $homeDir '.vrooli/bin')
}

function Get-VrooliArchitecture {
	$arch = $env:PROCESSOR_ARCHITECTURE
	if ([Environment]::Is64BitOperatingSystem -and $env:PROCESSOR_ARCHITEW6432) {
		$arch = $env:PROCESSOR_ARCHITEW6432
	}
	switch -Regex ($arch) {
		'^(AMD64|x86_64)$' { return 'amd64' }
		'^(ARM64|aarch64)$' { return 'arm64' }
		default { throw "Unsupported Windows architecture: $arch" }
	}
}

function Invoke-VrooliDownload([string]$Uri, [string]$OutFile) {
	Invoke-WebRequest -UseBasicParsing -Uri $Uri -OutFile $OutFile
}

function Get-VrooliExpectedChecksum([string]$Manifest, [string]$Asset) {
	foreach ($line in Get-Content -LiteralPath $Manifest) {
		$parts = $line -split '\s+', 2
		if ($parts.Count -eq 2 -and $parts[1].TrimStart('*') -eq $Asset) { return $parts[0] }
	}
	throw "Release checksum manifest does not contain $Asset"
}

function Test-VrooliChecksum([string]$Manifest, [string]$Asset, [string]$Path) {
	$expected = Get-VrooliExpectedChecksum $Manifest $Asset
	$actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $Path).Hash.ToLowerInvariant()
	if ($actual -ne $expected.ToLowerInvariant()) {
		throw "Checksum mismatch for $Asset (expected $expected, got $actual)"
	}
}

function Test-VrooliManifestSignature([string]$Manifest, [string]$SignatureBase64) {
	# RSA public parameters for install/vrooli-release.pub. RSAParameters works
	# on Windows PowerShell 5.1 as well as modern PowerShell, without OpenSSL.
	$modulus = 'zTHpnCxVTVn7wSr/ZCplJ5U07JXI5BLwpXoz2HVljpD4Rxv3A++PQ3E6Yje0C/F7AKH6+KAToETwPxiFxOQZVuU9ZYJXDMOIxEvOqu0yBKaQC2o2sv+JTmoWCCmxf4ADk2JQ/zMvHZpy9tO0Z1Tg1dGORGV9UaKd7l5Lm7Aam0VtJ6NDXuzwbTb+Ag+98xiM88MJX1RcJVdM5auwExKgqmHsBiFIL48jRCaJYMDtk5uB7UkFw9QsqbG4PM2J7lGnf5UgyIcM7KXH5HU009rqapqFy9dt6BlAS4xByEpOB0uf7v79zdNqRiKPhC0Zu3rT4mfslhz9TQj2q1C/zfOHXflKh/DY9qvhlD6rXbRCrIf31PP1ZVzu4sryQxxrPEAfNsdns4mmmSSGwVkKIn+/dOCvbCEAy83Mxt5XN66Yn94rKO/8Z2V3qer7789Fq39fJC4fkbngFkLqpG9HxSl5xOWW4L6Cfs7uecjvPUcxOqWkcpFR3PTM9gcGdqonbagL'
	$rsa = New-Object System.Security.Cryptography.RSACryptoServiceProvider
	try {
		$params = New-Object System.Security.Cryptography.RSAParameters
		$params.Modulus = [Convert]::FromBase64String($modulus)
		$params.Exponent = [byte[]](1, 0, 1)
		$rsa.ImportParameters($params)
		$data = [IO.File]::ReadAllBytes($Manifest)
		$signature = [Convert]::FromBase64String((Get-Content -Raw -LiteralPath $SignatureBase64).Trim())
		if (-not $rsa.VerifyData($data, 'SHA256', $signature)) { throw 'Release signature verification failed.' }
	}
	finally { $rsa.Dispose() }
}
