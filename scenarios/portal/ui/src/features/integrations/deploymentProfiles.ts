export const DEPLOYMENT_PROFILE_STORAGE_KEY = "portal.deployment-profile.v1";

export type DeploymentProfile = "client" | "local-control" | "automation";

export interface DeploymentProfileConfig {
  readonly version: 1;
  readonly profile: DeploymentProfile;
  readonly endpoint: string;
}

export interface IntegrationAvailability {
  readonly id: string;
  readonly state: string;
}

export interface DeploymentProfileStatus {
  readonly profile: DeploymentProfile;
  readonly endpoint: string;
  readonly ready: boolean;
  readonly missingCapabilities: readonly string[];
}

export const DEPLOYMENT_PROFILES: readonly {
  readonly id: DeploymentProfile;
  readonly requiredCapabilities: readonly string[];
}[] = [
  { id: "client", requiredCapabilities: [] },
  { id: "local-control", requiredCapabilities: ["device-control"] },
  { id: "automation", requiredCapabilities: ["agent-manager"] },
];

const PROFILE_IDS = new Set<DeploymentProfile>(DEPLOYMENT_PROFILES.map(({ id }) => id));
const MAX_ENDPOINT_LENGTH = 2048;

function storageOrNull(storage?: Storage | null): Storage | null {
  if (storage !== undefined) return storage;
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

/**
 * Endpoints are configuration, never credentials. Reject userinfo, query and
 * fragment components so a copied token cannot be persisted accidentally.
 */
export function validateDeploymentEndpoint(value: string): string {
  const endpoint = value.trim();
  if (endpoint === "") return "";
  if (endpoint.length > MAX_ENDPOINT_LENGTH) throw new Error("endpoint is too long");
  let parsed: URL;
  try {
    parsed = new URL(endpoint);
  } catch {
    throw new Error("endpoint must be a valid URL");
  }
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("endpoint must use http or https");
  }
  if (parsed.username || parsed.password || parsed.search || parsed.hash) {
    throw new Error("endpoint cannot contain credentials or query data");
  }
  return parsed.toString().replace(/\/$/, "");
}

export function defaultDeploymentProfileConfig(): DeploymentProfileConfig {
  return { version: 1, profile: "client", endpoint: "" };
}

export function loadDeploymentProfileConfig(storage?: Storage | null): DeploymentProfileConfig {
  const target = storageOrNull(storage);
  if (!target) return defaultDeploymentProfileConfig();
  try {
    const raw = target.getItem(DEPLOYMENT_PROFILE_STORAGE_KEY);
    if (!raw) return defaultDeploymentProfileConfig();
    const value: unknown = JSON.parse(raw);
    if (!value || typeof value !== "object") return defaultDeploymentProfileConfig();
    const candidate = value as Partial<DeploymentProfileConfig>;
    if (candidate.version !== 1 || typeof candidate.profile !== "string" || !PROFILE_IDS.has(candidate.profile) || typeof candidate.endpoint !== "string") {
      return defaultDeploymentProfileConfig();
    }
    return {
      version: 1,
      profile: candidate.profile,
      endpoint: validateDeploymentEndpoint(candidate.endpoint),
    };
  } catch {
    return defaultDeploymentProfileConfig();
  }
}

export function saveDeploymentProfileConfig(config: Omit<DeploymentProfileConfig, "version">, storage?: Storage | null): DeploymentProfileConfig {
  if (!PROFILE_IDS.has(config.profile)) throw new Error("unknown deployment profile");
  const normalized: DeploymentProfileConfig = {
    version: 1,
    profile: config.profile,
    endpoint: validateDeploymentEndpoint(config.endpoint),
  };
  const target = storageOrNull(storage);
  if (target) target.setItem(DEPLOYMENT_PROFILE_STORAGE_KEY, JSON.stringify(normalized));
  return normalized;
}

export function evaluateDeploymentProfile(
  config: DeploymentProfileConfig,
  integrations: readonly IntegrationAvailability[],
): DeploymentProfileStatus {
  const definition = DEPLOYMENT_PROFILES.find(({ id }) => id === config.profile) ?? DEPLOYMENT_PROFILES[0];
  const available = new Set(integrations.filter((item) => item.state === "available").map((item) => item.id));
  const missingCapabilities = (definition?.requiredCapabilities ?? []).filter((id) => !available.has(id));
  return {
    profile: config.profile,
    endpoint: config.endpoint,
    ready: missingCapabilities.length === 0,
    missingCapabilities,
  };
}
