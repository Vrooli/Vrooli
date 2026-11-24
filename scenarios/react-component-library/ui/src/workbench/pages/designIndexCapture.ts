import type { TFunction } from "i18next";

import { MAX_WORKSPACE_ASSET_BYTES } from "../../api/workspaces";
import { strings } from "../../consts/strings";

export type UxCaptureManifest = {
  schema_version: string;
  id: string;
  captured_at: string;
  source: { kind: string; scenario?: string; route?: string; build_id?: string; revision?: string };
  session: { id: string };
  viewport: { width: number; height: number };
  lenses: Record<string, { status: string; reason?: string }>;
  artifacts: Array<{
    kind: string;
    path: string;
    media_type: string;
    size_bytes: number;
    sha256: string;
  }>;
  privacy: { redaction_status: string; external?: boolean; ephemeral?: boolean };
};

type JsonRecord = Record<string, unknown>;

function isJsonRecord(value: unknown): value is JsonRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isString(value: unknown): value is string {
  return typeof value === "string";
}

export async function validateCapture(
  manifestFile: File,
  artifactFiles: File[],
  t: TFunction,
): Promise<{ manifest: UxCaptureManifest; pixels: File; frozenPage: File }> {
  const parsedManifest: unknown = JSON.parse(await readTextFile(manifestFile, t));
  if (!isJsonRecord(parsedManifest)) {
    throw new Error(t(strings.workbench.designIndex.captureInvalidManifest));
  }
  const sourceValue = parsedManifest.source;
  const sessionValue = parsedManifest.session;
  const viewportValue = parsedManifest.viewport;
  const privacyValue = parsedManifest.privacy;
  const lensesValue = parsedManifest.lenses;
  const artifactsValue = parsedManifest.artifacts;
  if (
    parsedManifest.schema_version !== "ux-capture.v1" ||
    !isString(parsedManifest.id) ||
    !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(parsedManifest.id) ||
    !isString(parsedManifest.captured_at) ||
    !isJsonRecord(sourceValue) ||
    !isJsonRecord(sessionValue) ||
    !isJsonRecord(viewportValue) ||
    !isJsonRecord(privacyValue) ||
    !isJsonRecord(lensesValue) ||
    !Array.isArray(artifactsValue)
  ) {
    throw new Error(t(strings.workbench.designIndex.captureInvalidManifest));
  }
  if (
    sourceValue.kind !== "scenario" ||
    !isString(sourceValue.scenario) ||
    !sourceValue.scenario ||
    !isString(sourceValue.route) ||
    !sourceValue.route ||
    !isString(sourceValue.build_id) ||
    !sourceValue.build_id ||
    (sourceValue.revision !== undefined && !isString(sourceValue.revision))
  ) {
    throw new Error(t(strings.workbench.designIndex.captureScenarioRequired));
  }
  if (
    !isString(sessionValue.id) ||
    !sessionValue.id ||
    !Number.isFinite(Date.parse(parsedManifest.captured_at)) ||
    !Number.isSafeInteger(viewportValue.width) ||
    !Number.isSafeInteger(viewportValue.height) ||
    typeof viewportValue.width !== "number" ||
    typeof viewportValue.height !== "number" ||
    viewportValue.width <= 0 ||
    viewportValue.height <= 0
  ) {
    throw new Error(t(strings.workbench.designIndex.captureViewportRequired));
  }
  if (privacyValue.redaction_status !== "complete") {
    throw new Error(t(strings.workbench.designIndex.captureRedactionRequired));
  }
  if (
    (privacyValue.external !== undefined && typeof privacyValue.external !== "boolean") ||
    (privacyValue.ephemeral !== undefined && typeof privacyValue.ephemeral !== "boolean")
  ) {
    throw new Error(t(strings.workbench.designIndex.captureInvalidManifest));
  }
  const lenses: UxCaptureManifest["lenses"] = {};
  for (const lens of ["L0", "L1", "L2", "L3", "L4", "L5", "L6"]) {
    const lensValue = lensesValue[lens];
    if (
      !isJsonRecord(lensValue) ||
      !isString(lensValue.status) ||
      !["present", "partial", "absent"].includes(lensValue.status) ||
      (lensValue.reason !== undefined && !isString(lensValue.reason)) ||
      (lensValue.status !== "present" && !lensValue.reason?.trim())
    ) {
      throw new Error(t(strings.workbench.designIndex.captureLensInvalid, { lens }));
    }
    lenses[lens] = {
      status: lensValue.status,
      ...(isString(lensValue.reason) ? { reason: lensValue.reason } : {}),
    };
  }
  const artifacts: UxCaptureManifest["artifacts"] = artifactsValue.map((artifactValue) => {
    if (
      !isJsonRecord(artifactValue) ||
      !isString(artifactValue.kind) ||
      !isString(artifactValue.path) ||
      !isString(artifactValue.media_type) ||
      typeof artifactValue.size_bytes !== "number" ||
      !isString(artifactValue.sha256)
    ) {
      throw new Error(t(strings.workbench.designIndex.captureArtifactInvalid));
    }
    return {
      kind: artifactValue.kind,
      path: artifactValue.path,
      media_type: artifactValue.media_type,
      size_bytes: artifactValue.size_bytes,
      sha256: artifactValue.sha256,
    };
  });
  const manifest: UxCaptureManifest = {
    schema_version: "ux-capture.v1",
    id: parsedManifest.id,
    captured_at: parsedManifest.captured_at,
    source: {
      kind: "scenario",
      scenario: sourceValue.scenario,
      route: sourceValue.route,
      build_id: sourceValue.build_id,
      ...(isString(sourceValue.revision) ? { revision: sourceValue.revision } : {}),
    },
    session: { id: sessionValue.id },
    viewport: { width: viewportValue.width, height: viewportValue.height },
    privacy: {
      redaction_status: "complete",
      ...(typeof privacyValue.external === "boolean" ? { external: privacyValue.external } : {}),
      ...(typeof privacyValue.ephemeral === "boolean" ? { ephemeral: privacyValue.ephemeral } : {}),
    },
    lenses,
    artifacts,
  };
  const l0 = manifest.lenses.L0;
  if (!l0 || l0.status !== "present")
    throw new Error(t(strings.workbench.designIndex.captureL0Required));
  const l1 = manifest.lenses.L1;
  if (!l1 || l1.status !== "present")
    throw new Error(t(strings.workbench.designIndex.captureL1Required));
  if (!Array.isArray(manifest.artifacts) || manifest.artifacts.length === 0)
    throw new Error(t(strings.workbench.designIndex.captureArtifactsRequired));
  const declaredPaths = new Set<string>();
  const declaredFileNames = new Set<string>();
  const uploadedByName = new Map<string, File>();
  for (const file of artifactFiles) {
    if (uploadedByName.has(file.name))
      throw new Error(
        t(strings.workbench.designIndex.captureDuplicateArtifact, { name: file.name }),
      );
    uploadedByName.set(file.name, file);
  }
  if (artifactFiles.reduce((total, file) => total + file.size, 0) > 64 * 1024 * 1024)
    throw new Error(t(strings.workbench.designIndex.captureArtifactLimit));
  let pixels: File | undefined;
  let frozenPage: File | undefined;
  for (const artifact of manifest.artifacts) {
    if (
      artifact.path.startsWith("/") ||
      /^[a-z][a-z0-9+.-]*:/i.test(artifact.path) ||
      artifact.path.split(/[\\/]/).some((part) => !part || part === "." || part === "..") ||
      !Number.isSafeInteger(artifact.size_bytes) ||
      artifact.size_bytes < 0 ||
      !/^[0-9a-f]{64}$/i.test(artifact.sha256)
    ) {
      throw new Error(t(strings.workbench.designIndex.captureArtifactInvalid));
    }
    if (declaredPaths.has(artifact.path))
      throw new Error(t(strings.workbench.designIndex.captureDuplicatePath));
    declaredPaths.add(artifact.path);
    const fileName = artifact.path.split(/[\\/]/).at(-1) ?? "";
    if (declaredFileNames.has(fileName))
      throw new Error(t(strings.workbench.designIndex.captureDuplicatePath));
    declaredFileNames.add(fileName);
    const file = uploadedByName.get(fileName);
    if (!file || file.size !== artifact.size_bytes)
      throw new Error(
        t(strings.workbench.designIndex.captureArtifactMissing, { path: artifact.path }),
      );
    const digest = new Uint8Array(await crypto.subtle.digest("SHA-256", await file.arrayBuffer()));
    const actual = Array.from(digest, (byte) => byte.toString(16).padStart(2, "0")).join("");
    if (actual !== artifact.sha256.toLowerCase())
      throw new Error(
        t(strings.workbench.designIndex.captureDigestMismatch, { path: artifact.path }),
      );
    if (artifact.kind === "pixels") {
      if (pixels) throw new Error(t(strings.workbench.designIndex.capturePixelsDuplicate));
      if (artifact.media_type !== "image/png")
        throw new Error(t(strings.workbench.designIndex.capturePngRequired));
      pixels = file;
    }
    if (artifact.kind === "frozen-page") {
      if (frozenPage) throw new Error(t(strings.workbench.designIndex.captureFrozenPageDuplicate));
      if (artifact.media_type !== "text/html" || !/\.html?$/i.test(file.name)) {
        throw new Error(t(strings.workbench.designIndex.captureFrozenPageRequired));
      }
      frozenPage = file;
    }
  }
  if (uploadedByName.size !== manifest.artifacts.length)
    throw new Error(t(strings.workbench.designIndex.captureUnexpectedArtifacts));
  if (!pixels || pixels.type !== "image/png")
    throw new Error(t(strings.workbench.designIndex.capturePngRequired));
  if (!frozenPage) throw new Error(t(strings.workbench.designIndex.captureFrozenPageRequired));
  if (pixels.size > MAX_WORKSPACE_ASSET_BYTES)
    throw new Error(
      t(strings.workbench.designIndex.capturePixelLimit, {
        limit: MAX_WORKSPACE_ASSET_BYTES / 1024 / 1024,
      }),
    );
  return { manifest, pixels, frozenPage };
}

export function readTextFile(file: File, t: TFunction): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(typeof reader.result === "string" ? reader.result : "");
    reader.onerror = () =>
      reject(reader.error ?? new Error(t(strings.workbench.designIndex.captureReadFailed)));
    reader.readAsText(file);
  });
}
