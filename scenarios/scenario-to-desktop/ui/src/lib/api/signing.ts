import { buildUrl, throwIfNotOk } from "./client";
import type {
  SigningConfig,
  SigningConfigResponse,
  SigningValidationResult,
  SigningReadinessResponse,
  ToolDetectionResult,
  DiscoveredCertificate,
  GenerateKeyResponse,
  WindowsSigningConfig,
  MacOSSigningConfig,
  LinuxSigningConfig,
} from "./types";

export async function fetchSigningConfig(scenario: string): Promise<SigningConfigResponse> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}`));
  await throwIfNotOk(response);
  return await response.json() as SigningConfigResponse;
}

export async function saveSigningConfig(scenario: string, config: SigningConfig): Promise<SigningConfigResponse> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config)
  });
  await throwIfNotOk(response);
  return await response.json() as SigningConfigResponse;
}

export async function updatePlatformSigningConfig(
  scenario: string,
  platform: "windows" | "macos" | "linux",
  config: WindowsSigningConfig | MacOSSigningConfig | LinuxSigningConfig
): Promise<SigningConfigResponse> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}/${platform}`), {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config)
  });
  await throwIfNotOk(response);
  return await response.json() as SigningConfigResponse;
}

export async function deleteSigningConfig(scenario: string): Promise<{ status: string; scenario: string }> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}`), {
    method: "DELETE"
  });
  await throwIfNotOk(response);
  return await response.json() as { status: string; scenario: string };
}

export async function deletePlatformSigningConfig(
  scenario: string,
  platform: "windows" | "macos" | "linux"
): Promise<{ status: string; scenario: string; platform: string }> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}/${platform}`), {
    method: "DELETE"
  });
  await throwIfNotOk(response);
  return await response.json() as { status: string; scenario: string; platform: string };
}

export async function validateSigningConfig(scenario: string): Promise<SigningValidationResult> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}/validate`), {
    method: "POST"
  });
  await throwIfNotOk(response);
  return await response.json() as SigningValidationResult;
}

export async function checkSigningReadiness(scenario: string): Promise<SigningReadinessResponse> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}/ready`));
  await throwIfNotOk(response);
  return await response.json() as SigningReadinessResponse;
}

export async function fetchSigningPrerequisites(): Promise<{ tools: ToolDetectionResult[] }> {
  const response = await fetch(buildUrl("/signing/prerequisites"));
  await throwIfNotOk(response);
  return await response.json() as { tools: ToolDetectionResult[] };
}

export async function discoverCertificates(platform: "windows" | "macos" | "linux"): Promise<{
  platform: string;
  certificates: DiscoveredCertificate[];
}> {
  const response = await fetch(buildUrl(`/signing/discover/${platform}`));
  await throwIfNotOk(response);
  return await response.json() as { platform: string; certificates: DiscoveredCertificate[] };
}

export async function generateLinuxSigningKey(
  scenario: string,
  payload: {
    name?: string;
    email?: string;
    passphrase?: string;
    passphrase_env?: string;
    homedir?: string;
    expiry?: string;
    force?: boolean;
  }
): Promise<GenerateKeyResponse> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}/linux/generate-key`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ...payload, export_public: true })
  });
  await throwIfNotOk(response);
  return await response.json() as GenerateKeyResponse;
}
