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
	$modulus = 'nF9P/cpLxBCwyN50qW1x63SB2/Ngn5HwuU/p+Upo0BUy9aT2tTXL6tZQw8q/8UTYZdj2pn7IoB89G+6XPerbSi/93jYjxoV9xilQHdjp5dcnsEpDcvC6v3JQj/JLV+HEoRyQ7qAseVCAELmBi5qHy+Vm3naH7Z/a36+vcePHSMDoPeoE0Q7pEEPR0U7LQX7tQJOAHvd0b87SExtuBxa3OXY7Mxw/rn7FBcEM4ZWMwBn0vAVRts0ZOfBF5KLw5u5ouBpNUVYT8WNn3JNmPcKARIyF841rVXxU79u70LkJGU3NZncOlZjJyViQJZTLMAFQEYvjZQ2w7CEW6qAOs6HLNmIC3lUl3Z35g0HqKyseZZ2vMllFQUBKvw7ui3bzEhl6KWEx8EpGpC41k3BjZpH2I0pUGh74bba7w5XniW5fuCwEZ2qduwzx5SLNiioHDvrDzm7FZHsP72Mf3MPeoPEUxIcw+wH0L4LiBzZSzI2oGeiBJKGYZnRtQjlw0wLFM+Ah'
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
