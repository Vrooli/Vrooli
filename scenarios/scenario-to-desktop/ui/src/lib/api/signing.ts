import { signingConnectClient } from "./connect";
import {
  presentDiscoveredCertificates,
  presentSigningPrerequisites,
  presentSigningReadiness,
  presentSigningValidation,
  signingConfigFromProto,
  signingConfigToProto,
  signingPlatformToProto,
  windowsSigningConfigToProto,
  macosSigningConfigToProto,
  linuxSigningConfigToProto,
  type LinuxSigningConfig,
  type MacOSSigningConfig,
  type SigningConfig,
  type SigningPlatform,
  type WindowsSigningConfig,
} from "../../domain/signing";

export async function fetchSigningConfig(scenarioName: string) {
  const response = await signingConnectClient.getSigningConfig({
    scenarioName,
  });
  return {
    config: response.config ? signingConfigFromProto(response.config) : null,
  };
}
export async function saveSigningConfig(
  scenarioName: string,
  config: SigningConfig,
) {
  const response = await signingConnectClient.putSigningConfig({
    scenarioName,
    config: signingConfigToProto(config),
  });
  return {
    config: response.config ? signingConfigFromProto(response.config) : null,
  };
}
export async function updatePlatformSigningConfig(
  scenarioName: string,
  platform: SigningPlatform,
  config: WindowsSigningConfig | MacOSSigningConfig | LinuxSigningConfig,
) {
  const response = await signingConnectClient.patchSigningPlatform({
    scenarioName,
    platform: signingPlatformToProto(platform),
    config:
      platform === "windows"
        ? {
            case: "windows",
            value: windowsSigningConfigToProto(config as WindowsSigningConfig),
          }
        : platform === "macos"
          ? {
              case: "macos",
              value: macosSigningConfigToProto(config as MacOSSigningConfig),
            }
          : {
              case: "linux",
              value: linuxSigningConfigToProto(config as LinuxSigningConfig),
            },
  });
  return {
    config: response.config ? signingConfigFromProto(response.config) : null,
  };
}
export async function deleteSigningConfig(scenarioName: string) {
  const response = await signingConnectClient.deleteSigningConfig({
    scenarioName,
  });
  return { status: "deleted", scenario: response.scenarioName };
}
export async function deletePlatformSigningConfig(
  scenarioName: string,
  platform: SigningPlatform,
) {
  const response = await signingConnectClient.deleteSigningPlatform({
    scenarioName,
    platform: signingPlatformToProto(platform),
  });
  return { status: "deleted", scenario: response.scenarioName, platform };
}
export async function validateSigningConfig(scenarioName: string) {
  const current = await signingConnectClient.getSigningConfig({ scenarioName });
  return presentSigningValidation(
    await signingConnectClient.validateSigningConfig({
      scenarioName,
      config: current.config,
    }),
  );
}
export async function checkSigningReadiness(scenarioName: string) {
  return presentSigningReadiness(
    await signingConnectClient.getSigningReadiness({ scenarioName }),
  );
}
export async function fetchSigningPrerequisites() {
  return {
    tools: presentSigningPrerequisites(
      await signingConnectClient.listSigningPrerequisites({}),
    ),
  };
}
export async function discoverCertificates(platform: SigningPlatform) {
  return {
    platform,
    certificates: presentDiscoveredCertificates(
      await signingConnectClient.discoverSigningCertificates({
        platform: signingPlatformToProto(platform),
      }),
    ),
  };
}
export async function generateLinuxSigningKey(
  scenarioName: string,
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
  if (payload.passphrase)
    throw new Error(
      "passphrase is not supported; provide passphrase_env instead",
    );
  const response = await signingConnectClient.generateLinuxSigningKey({
    scenarioName,
    name: payload.name,
    email: payload.email,
    passphraseEnv: payload.passphrase_env,
    homedir: payload.homedir,
    expiry: payload.expiry,
    force: payload.force,
    exportPublic: true,
  });
  return {
    key_id: response.keyId,
    fingerprint: response.fingerprint,
    homedir: response.homedir,
    public_key: response.publicKey,
    public_key_path: response.publicKeyPath,
  };
}
