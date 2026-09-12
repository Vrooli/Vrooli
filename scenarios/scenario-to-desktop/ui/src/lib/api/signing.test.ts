/* eslint-disable @typescript-eslint/no-unsafe-assignment -- Vitest asymmetric matchers erase type data. */
import { describe, expect, it, vi } from "vitest";
import {
  CertificateSource,
  SignAlgorithm,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/signing_pb";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

const client = vi.hoisted(() => ({
  getSigningConfig: vi.fn(),
  putSigningConfig: vi.fn(),
  patchSigningPlatform: vi.fn(),
  deleteSigningConfig: vi.fn(),
  deleteSigningPlatform: vi.fn(),
  validateSigningConfig: vi.fn(),
  getSigningReadiness: vi.fn(),
  listSigningPrerequisites: vi.fn(),
  discoverSigningCertificates: vi.fn(),
  generateLinuxSigningKey: vi.fn(),
}));
vi.mock("./connect", () => ({ signingConnectClient: client }));

import {
  checkSigningReadiness,
  deletePlatformSigningConfig,
  deleteSigningConfig,
  fetchSigningConfig,
  generateLinuxSigningKey,
  saveSigningConfig,
  updatePlatformSigningConfig,
  validateSigningConfig,
} from "./signing";

describe("signing Connect client", () => {
  it("serializes a complete signing configuration with generated enums", async () => {
    client.putSigningConfig.mockResolvedValue({ config: undefined });
    await expect(
      saveSigningConfig("calculator", {
        enabled: true,
        schema_version: "v1",
        windows: {
          certificate_source: "azure_keyvault",
          certificate_file: "/cert.pfx",
          certificate_password_env: "CERT_PASSWORD",
          sign_algorithm: "sha512",
        },
        macos: {
          identity: "Developer ID",
          team_id: "TEAM",
          hardened_runtime: true,
          notarize: true,
        },
        linux: {
          gpg_key_id: "ABC",
          gpg_passphrase_env: "GPG_PASSWORD",
          keyring_path: "/keyring",
        },
      }),
    ).resolves.toEqual({ config: null });

    expect(client.putSigningConfig).toHaveBeenCalledWith(
      expect.objectContaining({
        scenarioName: "calculator",
        config: expect.objectContaining({
          schemaVersion: "v1",
          windows: expect.objectContaining({
            certificateSource: CertificateSource.AZURE_KEY_VAULT,
            signAlgorithm: SignAlgorithm.SHA512,
          }),
          macos: expect.objectContaining({
            teamId: "TEAM",
            hardenedRuntime: true,
          }),
          linux: expect.objectContaining({
            gpgKeyId: "ABC",
            passphraseEnv: "GPG_PASSWORD",
          }),
        }),
      }),
    );
  });

  it("maps generated signing config and platform oneofs in both directions", async () => {
    client.getSigningConfig.mockResolvedValue({
      config: {
        enabled: true,
        schemaVersion: "v1",
        windows: {
          certificateSource: CertificateSource.STORE,
          signAlgorithm: SignAlgorithm.SHA384,
        },
      },
    });
    client.patchSigningPlatform.mockResolvedValue({ config: undefined });

    await expect(fetchSigningConfig("calculator")).resolves.toMatchObject({
      config: {
        enabled: true,
        schema_version: "v1",
        windows: { certificate_source: "store", sign_algorithm: "sha384" },
      },
    });
    await updatePlatformSigningConfig("calculator", "windows", {
      certificate_source: "file",
      certificate_file: "/cert.pfx",
      sign_algorithm: "sha256",
    });
    await updatePlatformSigningConfig("calculator", "macos", {
      identity: "Developer ID",
      team_id: "TEAM",
      hardened_runtime: true,
      notarize: false,
    });
    await updatePlatformSigningConfig("calculator", "linux", {
      gpg_key_id: "ABC",
    });

    expect(client.patchSigningPlatform).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        scenarioName: "calculator",
        platform: Platform.WIN,
        config: {
          case: "windows",
          value: expect.objectContaining({
            certificateSource: CertificateSource.FILE,
          }),
        },
      }),
    );
    expect(client.patchSigningPlatform).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        platform: Platform.MAC,
        config: {
          case: "macos",
          value: expect.objectContaining({ teamId: "TEAM" }),
        },
      }),
    );
    expect(client.patchSigningPlatform).toHaveBeenNthCalledWith(
      3,
      expect.objectContaining({
        platform: Platform.LINUX,
        config: {
          case: "linux",
          value: expect.objectContaining({ gpgKeyId: "ABC" }),
        },
      }),
    );
  });

  it("preserves generated status responses and rejects plaintext key passphrases", async () => {
    client.deleteSigningConfig.mockResolvedValue({
      scenarioName: "calculator",
    });
    client.deleteSigningPlatform.mockResolvedValue({
      scenarioName: "calculator",
      platform: Platform.MAC,
    });
    client.validateSigningConfig.mockResolvedValue({
      valid: false,
      errors: [{ code: "missing", field: "identity", message: "required" }],
      warnings: [],
    });
    client.getSigningReadiness.mockResolvedValue({
      ready: true,
      platforms: [{ platform: Platform.LINUX, ready: true, message: "ready" }],
    });
    client.generateLinuxSigningKey.mockResolvedValue({
      keyId: "key-1",
      fingerprint: "fingerprint",
    });

    await expect(deleteSigningConfig("calculator")).resolves.toEqual({
      status: "deleted",
      scenario: "calculator",
    });
    await expect(
      deletePlatformSigningConfig("calculator", "macos"),
    ).resolves.toEqual({
      status: "deleted",
      scenario: "calculator",
      platform: "macos",
    });
    await expect(validateSigningConfig("calculator")).resolves.toEqual({
      valid: false,
      errors: [{ code: "missing", field: "identity", message: "required" }],
      warnings: [],
    });
    await expect(checkSigningReadiness("calculator")).resolves.toEqual({
      ready: true,
      platforms: { linux: { ready: true, reason: "ready" } },
    });
    await expect(
      generateLinuxSigningKey("calculator", { passphrase: "never-send-this" }),
    ).rejects.toThrow("passphrase is not supported");
    await expect(
      generateLinuxSigningKey("calculator", {
        name: "Calculator",
        passphrase_env: "GPG_PASSWORD",
      }),
    ).resolves.toMatchObject({ key_id: "key-1" });
    expect(client.generateLinuxSigningKey).toHaveBeenCalledWith(
      expect.objectContaining({
        scenarioName: "calculator",
        passphraseEnv: "GPG_PASSWORD",
        exportPublic: true,
      }),
    );
  });
});
