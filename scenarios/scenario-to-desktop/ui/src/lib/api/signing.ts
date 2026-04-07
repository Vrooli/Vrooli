import { fetchJson, mutateJson } from "./client";
import type {
  WindowsSigningConfig,
  MacOSSigningConfig,
  LinuxSigningConfig,
  SigningConfig,
} from "./types";
import {
  SigningConfigResponseSchema,
  SigningValidationResultSchema,
  SigningReadinessResponseSchema,
  GenerateKeyResponseSchema,
  DeleteSigningResponseSchema,
  DeletePlatformSigningResponseSchema,
  PrerequisitesResponseSchema,
  CertificateDiscoveryResponseSchema,
} from "./schemas/signing";

export function fetchSigningConfig(scenario: string) {
  return fetchJson(
    `/signing/${encodeURIComponent(scenario)}`,
    SigningConfigResponseSchema,
  );
}

export function saveSigningConfig(scenario: string, config: SigningConfig) {
  return mutateJson(
    `/signing/${encodeURIComponent(scenario)}`,
    SigningConfigResponseSchema,
    { method: "PUT", body: config },
  );
}

export function updatePlatformSigningConfig(
  scenario: string,
  platform: "windows" | "macos" | "linux",
  config: WindowsSigningConfig | MacOSSigningConfig | LinuxSigningConfig,
) {
  return mutateJson(
    `/signing/${encodeURIComponent(scenario)}/${platform}`,
    SigningConfigResponseSchema,
    { method: "PATCH", body: config },
  );
}

export function deleteSigningConfig(scenario: string) {
  return mutateJson(
    `/signing/${encodeURIComponent(scenario)}`,
    DeleteSigningResponseSchema,
    { method: "DELETE" },
  );
}

export function deletePlatformSigningConfig(
  scenario: string,
  platform: "windows" | "macos" | "linux",
) {
  return mutateJson(
    `/signing/${encodeURIComponent(scenario)}/${platform}`,
    DeletePlatformSigningResponseSchema,
    { method: "DELETE" },
  );
}

export function validateSigningConfig(scenario: string) {
  return mutateJson(
    `/signing/${encodeURIComponent(scenario)}/validate`,
    SigningValidationResultSchema,
    { method: "POST" },
  );
}

export function checkSigningReadiness(scenario: string) {
  return fetchJson(
    `/signing/${encodeURIComponent(scenario)}/ready`,
    SigningReadinessResponseSchema,
  );
}

export function fetchSigningPrerequisites() {
  return fetchJson("/signing/prerequisites", PrerequisitesResponseSchema);
}

export function discoverCertificates(platform: "windows" | "macos" | "linux") {
  return fetchJson(
    `/signing/discover/${platform}`,
    CertificateDiscoveryResponseSchema,
  );
}

export function generateLinuxSigningKey(
  scenario: string,
  payload: {
    name?: string;
    email?: string;
    passphrase?: string;
    passphrase_env?: string;
    homedir?: string;
    expiry?: string;
    force?: boolean;
  },
) {
  return mutateJson(
    `/signing/${encodeURIComponent(scenario)}/linux/generate-key`,
    GenerateKeyResponseSchema,
    { method: "POST", body: { ...payload, export_public: true } },
  );
}
