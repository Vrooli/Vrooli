/**
 * Zod schemas for code-signing API response types.
 *
 * Covers signing configuration CRUD, validation, readiness checks,
 * tool detection, certificate discovery, and key generation endpoints.
 */

import { z } from "zod";

// ==================== Platform Signing Configs ====================

export const WindowsSigningConfigSchema = z.object({
  certificate_source: z.enum(["file", "store", "azure_keyvault", "aws_kms"]),
  certificate_file: z.string().optional(),
  certificate_password_env: z.string().optional(),
  certificate_thumbprint: z.string().optional(),
  timestamp_server: z.string().optional(),
  sign_algorithm: z.enum(["sha256", "sha384", "sha512"]).optional(),
  dual_sign: z.boolean().optional(),
});

export const MacOSSigningConfigSchema = z.object({
  identity: z.string(),
  team_id: z.string(),
  hardened_runtime: z.boolean(),
  notarize: z.boolean(),
  entitlements_file: z.string().optional(),
  provisioning_profile: z.string().optional(),
  gatekeeper_assess: z.boolean().optional(),
  apple_id_env: z.string().optional(),
  apple_id_password_env: z.string().optional(),
  apple_api_key_id: z.string().optional(),
  apple_api_key_file: z.string().optional(),
  apple_api_issuer_id: z.string().optional(),
});

export const LinuxSigningConfigSchema = z.object({
  gpg_key_id: z.string().optional(),
  gpg_passphrase_env: z.string().optional(),
  gpg_homedir: z.string().optional(),
  keyring_path: z.string().optional(),
  deb_keyring_path: z.string().optional(),
  rpm_keyring_path: z.string().optional(),
});

export const SigningConfigSchema = z.object({
  schema_version: z.string().optional(),
  enabled: z.boolean(),
  windows: WindowsSigningConfigSchema.optional(),
  macos: MacOSSigningConfigSchema.optional(),
  linux: LinuxSigningConfigSchema.optional(),
});

// ==================== Response Types ====================

export const SigningConfigResponseSchema = z.object({
  scenario: z.string().optional(),
  config: SigningConfigSchema.nullable().optional(),
  config_path: z.string().optional(),
});

export const PlatformValidationSchema = z.object({
  configured: z.boolean(),
  tool_installed: z.boolean().optional(),
  tool_path: z.string().optional(),
  tool_version: z.string().optional(),
  errors: z.array(z.string()),
  warnings: z.array(z.string()),
});

export const SigningValidationErrorSchema = z.object({
  code: z.string(),
  platform: z.string().optional(),
  field: z.string().optional(),
  message: z.string(),
  remediation: z.string().optional(),
});

export const SigningValidationWarningSchema = z.object({
  code: z.string(),
  platform: z.string().optional(),
  message: z.string(),
});

export const SigningValidationResultSchema = z.object({
  valid: z.boolean(),
  platforms: z.record(PlatformValidationSchema).optional(),
  errors: z.array(SigningValidationErrorSchema).optional(),
  warnings: z.array(SigningValidationWarningSchema).optional(),
});

export const PlatformStatusSchema = z.object({
  ready: z.boolean(),
  reason: z.string().optional(),
});

export const SigningReadinessResponseSchema = z.object({
  ready: z.boolean(),
  scenario: z.string().optional(),
  issues: z.array(z.string()).optional(),
  platforms: z.record(PlatformStatusSchema).optional(),
});

export const ToolDetectionResultSchema = z.object({
  platform: z.string().optional(),
  tool: z.string().optional(),
  installed: z.boolean().optional(),
  path: z.string().optional(),
  version: z.string().optional(),
  error: z.string().optional(),
  remediation: z.string().optional(),
  name: z.string().optional(),
});

export const DiscoveredCertificateSchema = z.object({
  id: z.string().optional(),
  name: z.string().optional(),
  subject: z.string().optional(),
  issuer: z.string().optional(),
  expires_at: z.string().optional(),
  days_to_expiry: z.number().optional(),
  is_expired: z.boolean().optional(),
  is_code_sign: z.boolean().optional(),
  type: z.string().optional(),
  platform: z.string().optional(),
  usage_hint: z.string().optional(),
});

export const GenerateKeyResponseSchema = z.object({
  status: z.string().optional(),
  key_id: z.string().optional(),
  fingerprint: z.string().optional(),
  homedir: z.string().optional(),
  public_key: z.string().optional(),
  config_path: z.string().optional(),
  public_key_path: z.string().optional(),
});

// ==================== Inline response types ====================

export const DeleteSigningResponseSchema = z.object({
  status: z.string(),
  scenario: z.string(),
});

export const DeletePlatformSigningResponseSchema = z.object({
  status: z.string(),
  scenario: z.string(),
  platform: z.string(),
});

export const PrerequisitesResponseSchema = z.object({
  tools: z.array(ToolDetectionResultSchema),
});

export const CertificateDiscoveryResponseSchema = z.object({
  platform: z.string(),
  certificates: z.array(DiscoveredCertificateSchema),
});
