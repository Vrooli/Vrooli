import {
  CertificateSource,
  SignAlgorithm,
  type SigningConfig as ProtoSigningConfig,
  type ReadinessResponse,
  type ListSigningPrerequisitesResponse,
  type DiscoverSigningCertificatesResponse,
  type SigningValidationResult as ProtoSigningValidationResult,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/signing_pb";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

/** Editable UI state. It is intentionally separate from the generated wire model. */
export interface SigningConfig {
  schema_version?: string;
  enabled: boolean;
  windows?: WindowsSigningConfig;
  macos?: MacOSSigningConfig;
  linux?: LinuxSigningConfig;
}
export interface WindowsSigningConfig {
  certificate_source: "file" | "store" | "azure_keyvault" | "aws_kms";
  certificate_file?: string;
  certificate_password_env?: string;
  certificate_thumbprint?: string;
  timestamp_server?: string;
  sign_algorithm?: "sha256" | "sha384" | "sha512";
  dual_sign?: boolean;
}
export interface MacOSSigningConfig {
  identity: string;
  team_id: string;
  hardened_runtime: boolean;
  notarize: boolean;
  entitlements_file?: string;
  provisioning_profile?: string;
  gatekeeper_assess?: boolean;
  apple_id_env?: string;
  apple_id_password_env?: string;
  apple_api_key_id?: string;
  apple_api_key_file?: string;
  apple_api_issuer_id?: string;
}
export interface LinuxSigningConfig {
  gpg_key_id?: string;
  gpg_passphrase_env?: string;
  gpg_homedir?: string;
  keyring_path?: string;
  deb_keyring_path?: string;
  rpm_keyring_path?: string;
}
export type SigningPlatform = "windows" | "macos" | "linux";

const platformToProto: Record<SigningPlatform, Platform> = {
  windows: Platform.WIN,
  macos: Platform.MAC,
  linux: Platform.LINUX,
};
export const signingPlatformToProto = (platform: SigningPlatform) =>
  platformToProto[platform];
export function signingPlatformFromProto(
  platform: Platform,
): SigningPlatform | undefined {
  switch (platform) {
    case Platform.WIN:
      return "windows";
    case Platform.MAC:
      return "macos";
    case Platform.LINUX:
      return "linux";
  }
}

const certificateSourceToProto: Record<
  WindowsSigningConfig["certificate_source"],
  CertificateSource
> = {
  file: CertificateSource.FILE,
  store: CertificateSource.STORE,
  azure_keyvault: CertificateSource.AZURE_KEY_VAULT,
  aws_kms: CertificateSource.AWS_KMS,
};
const certificateSourceFromProto: Record<
  CertificateSource,
  WindowsSigningConfig["certificate_source"]
> = {
  [CertificateSource.UNSPECIFIED]: "file",
  [CertificateSource.FILE]: "file",
  [CertificateSource.STORE]: "store",
  [CertificateSource.AZURE_KEY_VAULT]: "azure_keyvault",
  [CertificateSource.AWS_KMS]: "aws_kms",
};
const signAlgorithmToProto: Record<
  NonNullable<WindowsSigningConfig["sign_algorithm"]>,
  SignAlgorithm
> = {
  sha256: SignAlgorithm.SHA256,
  sha384: SignAlgorithm.SHA384,
  sha512: SignAlgorithm.SHA512,
};
const signAlgorithmFromProto: Record<
  SignAlgorithm,
  NonNullable<WindowsSigningConfig["sign_algorithm"]>
> = {
  [SignAlgorithm.UNSPECIFIED]: "sha256",
  [SignAlgorithm.SHA256]: "sha256",
  [SignAlgorithm.SHA384]: "sha384",
  [SignAlgorithm.SHA512]: "sha512",
};

export function signingConfigToProto(config: SigningConfig) {
  return {
    enabled: config.enabled,
    schemaVersion: config.schema_version,
    windows: config.windows && {
      enabled: true,
      certificateSource:
        certificateSourceToProto[config.windows.certificate_source],
      certificatePath: config.windows.certificate_file,
      certificateThumbprint: config.windows.certificate_thumbprint,
      passwordEnv: config.windows.certificate_password_env,
      timestampServer: config.windows.timestamp_server,
      signAlgorithm: config.windows.sign_algorithm
        ? signAlgorithmToProto[config.windows.sign_algorithm]
        : SignAlgorithm.SHA256,
    },
    macos: config.macos && {
      enabled: true,
      identity: config.macos.identity,
      teamId: config.macos.team_id,
      hardenedRuntime: config.macos.hardened_runtime,
      notarize: config.macos.notarize,
      entitlementsPath: config.macos.entitlements_file,
      provisioningProfile: config.macos.provisioning_profile,
      appleId: config.macos.apple_id_env,
      applePasswordEnv: config.macos.apple_id_password_env,
    },
    linux: config.linux && {
      enabled: true,
      gpgKeyId: config.linux.gpg_key_id,
      passphraseEnv: config.linux.gpg_passphrase_env,
      keyringPath: config.linux.keyring_path,
    },
  };
}
export function signingConfigFromProto(
  config: ProtoSigningConfig,
): SigningConfig {
  return {
    enabled: config.enabled,
    schema_version: config.schemaVersion,
    windows: config.windows && {
      certificate_source:
        certificateSourceFromProto[config.windows.certificateSource],
      certificate_file: config.windows.certificatePath,
      certificate_password_env: config.windows.passwordEnv,
      certificate_thumbprint: config.windows.certificateThumbprint,
      timestamp_server: config.windows.timestampServer,
      sign_algorithm: signAlgorithmFromProto[config.windows.signAlgorithm],
    },
    macos: config.macos && {
      identity: config.macos.identity ?? "",
      team_id: config.macos.teamId ?? "",
      hardened_runtime: config.macos.hardenedRuntime ?? true,
      notarize: config.macos.notarize,
      entitlements_file: config.macos.entitlementsPath,
      provisioning_profile: config.macos.provisioningProfile,
      apple_id_env: config.macos.appleId,
      apple_id_password_env: config.macos.applePasswordEnv,
    },
    linux: config.linux && {
      gpg_key_id: config.linux.gpgKeyId,
      gpg_passphrase_env: config.linux.passphraseEnv,
      keyring_path: config.linux.keyringPath,
    },
  };
}

export function windowsSigningConfigToProto(config: WindowsSigningConfig) {
  return {
    enabled: true,
    certificateSource: certificateSourceToProto[config.certificate_source],
    certificatePath: config.certificate_file,
    certificateThumbprint: config.certificate_thumbprint,
    passwordEnv: config.certificate_password_env,
    timestampServer: config.timestamp_server,
    signAlgorithm: config.sign_algorithm
      ? signAlgorithmToProto[config.sign_algorithm]
      : SignAlgorithm.SHA256,
  };
}
export function macosSigningConfigToProto(config: MacOSSigningConfig) {
  return {
    enabled: true,
    identity: config.identity,
    teamId: config.team_id,
    hardenedRuntime: config.hardened_runtime,
    notarize: config.notarize,
    entitlementsPath: config.entitlements_file,
    provisioningProfile: config.provisioning_profile,
    appleId: config.apple_id_env,
    applePasswordEnv: config.apple_id_password_env,
  };
}
export function linuxSigningConfigToProto(config: LinuxSigningConfig) {
  return {
    enabled: true,
    gpgKeyId: config.gpg_key_id,
    passphraseEnv: config.gpg_passphrase_env,
    keyringPath: config.keyring_path,
  };
}

export interface SigningValidationError {
  code: string;
  platform?: string;
  field?: string;
  message: string;
  remediation?: string;
}
export interface ValidationWarning {
  code: string;
  platform?: string;
  message: string;
}
export interface PlatformValidation {
  configured: boolean;
  tool_installed?: boolean;
  tool_path?: string;
  tool_version?: string;
  errors: string[];
  warnings: string[];
}
export interface SigningValidationResult {
  valid: boolean;
  platforms?: Record<string, PlatformValidation>;
  errors: SigningValidationError[];
  warnings: ValidationWarning[];
}
export const presentSigningValidation = (
  result: ProtoSigningValidationResult,
): SigningValidationResult => ({
  valid: result.valid,
  errors: result.errors.map((error) => ({
    code: error.code,
    field: error.field,
    message: error.message,
  })),
  warnings: result.warnings.map((warning) => ({
    code: warning.code,
    message: warning.message,
  })),
});
export interface PlatformStatus {
  ready: boolean;
  reason?: string;
}
export interface SigningReadinessResponse {
  ready: boolean;
  scenario?: string;
  issues?: string[];
  platforms?: Record<string, PlatformStatus>;
}
export interface SigningConfigResponse {
  scenario?: string;
  config?: SigningConfig | null;
  config_path?: string;
}
export interface ToolDetectionResult {
  platform?: SigningPlatform | "all";
  tool?: string;
  installed?: boolean;
  path?: string;
  version?: string;
  error?: string;
  remediation?: string;
  name?: string;
}
export interface DiscoveredCertificate {
  id?: string;
  name?: string;
  subject?: string;
  issuer?: string;
  expires_at?: string;
  days_to_expiry?: number;
  is_expired?: boolean;
  is_code_sign?: boolean;
  type?: string;
  platform?: SigningPlatform;
  usage_hint?: string;
}
export interface GenerateKeyResponse {
  status?: string;
  key_id?: string;
  fingerprint?: string;
  homedir?: string;
  public_key?: string;
  config_path?: string;
  public_key_path?: string;
}

export function presentSigningReadiness(
  response: ReadinessResponse,
): SigningReadinessResponse {
  return {
    ready: response.ready,
    issues: response.message ? [response.message] : undefined,
    platforms: Object.fromEntries(
      response.platforms.flatMap((status) => {
        const platform = signingPlatformFromProto(status.platform);
        return platform
          ? [[platform, { ready: status.ready, reason: status.message }]]
          : [];
      }),
    ),
  };
}
export function presentSigningPrerequisites(
  response: ListSigningPrerequisitesResponse,
): ToolDetectionResult[] {
  return response.tools.map((tool) => ({
    platform: signingPlatformFromProto(tool.platform),
    tool: tool.tool,
    installed: tool.installed,
    path: tool.path,
    version: tool.version,
    error: tool.diagnostic,
    remediation: tool.remediation,
  }));
}
export function presentDiscoveredCertificates(
  response: DiscoverSigningCertificatesResponse,
): DiscoveredCertificate[] {
  return response.certificates.map((certificate) => ({
    id: certificate.id,
    name: certificate.name,
    subject: certificate.subject,
    issuer: certificate.issuer,
    expires_at: certificate.expiresAt,
    days_to_expiry: certificate.daysToExpiry,
    is_expired: certificate.expired,
    is_code_sign: certificate.codeSigning,
    type: certificate.type,
    platform: signingPlatformFromProto(certificate.platform),
    usage_hint: certificate.usageHint,
  }));
}
