import type { DeploymentManifestResponse, DeploymentManifestSecret } from "../../lib/api";
import type { ResourceGroup, FilterMode, ManifestSummary, SecretIdentifier } from "./types";
import { secretIdToString } from "./types";

/**
 * Groups secrets by resource name for tree display
 */
export function groupSecretsByResource(
  secrets: DeploymentManifestSecret[],
  excludedResources: Set<string>,
  excludedSecrets: Set<string>,
  overriddenSecrets: Set<string>
): ResourceGroup[] {
  const groups = new Map<string, DeploymentManifestSecret[]>();

  for (const secret of secrets) {
    const existing = groups.get(secret.resource_name) ?? [];
    existing.push(secret);
    groups.set(secret.resource_name, existing);
  }

  const result: ResourceGroup[] = [];
  for (const [resourceName, resourceSecrets] of groups) {
    const isResourceExcluded = excludedResources.has(resourceName);
    const strategizedCount = resourceSecrets.filter(
      (s) => s.handling_strategy && s.handling_strategy !== "none" && s.handling_strategy !== ""
    ).length;
    const blockingCount = resourceSecrets.filter(
      (s) => !s.handling_strategy || s.handling_strategy === "none" || s.handling_strategy === ""
    ).length;
    const allSecretsExcluded = resourceSecrets.every(
      (s) => excludedSecrets.has(secretIdToString({ resource: s.resource_name, key: s.secret_key }))
    );

    result.push({
      resourceName,
      secrets: resourceSecrets,
      strategizedCount,
      totalCount: resourceSecrets.length,
      blockingCount,
      hasBlockers: blockingCount > 0,
      isFullyExcluded: isResourceExcluded || allSecretsExcluded
    });
  }

  return result.sort((a, b) => a.resourceName.localeCompare(b.resourceName));
}

/**
 * Filters resource groups based on filter mode
 */
export function filterResourceGroups(
  groups: ResourceGroup[],
  filter: FilterMode,
  excludedResources: Set<string>,
  excludedSecrets: Set<string>,
  overriddenSecrets: Set<string>,
  searchQuery: string
): ResourceGroup[] {
  let filtered = groups;

  // Apply search filter
  if (searchQuery.trim()) {
    const query = searchQuery.toLowerCase();
    filtered = filtered.filter(
      (group) =>
        group.resourceName.toLowerCase().includes(query) ||
        group.secrets.some((s) => s.secret_key.toLowerCase().includes(query))
    );
  }

  // Apply filter mode
  switch (filter) {
    case "blocking":
      filtered = filtered.filter((group) => group.hasBlockers);
      break;
    case "overridden":
      filtered = filtered.filter((group) =>
        group.secrets.some((s) =>
          overriddenSecrets.has(secretIdToString({ resource: s.resource_name, key: s.secret_key }))
        )
      );
      break;
    case "excluded":
      filtered = filtered.filter(
        (group) =>
          excludedResources.has(group.resourceName) ||
          group.secrets.some((s) =>
            excludedSecrets.has(secretIdToString({ resource: s.resource_name, key: s.secret_key }))
          )
      );
      break;
    case "all":
    default:
      break;
  }

  return filtered;
}

/**
 * Filters secrets within a resource group based on filter mode and search
 */
export function filterSecretsInGroup(
  secrets: DeploymentManifestSecret[],
  filter: FilterMode,
  excludedSecrets: Set<string>,
  overriddenSecrets: Set<string>,
  searchQuery: string
): DeploymentManifestSecret[] {
  let filtered = secrets;

  // Apply search filter
  if (searchQuery.trim()) {
    const query = searchQuery.toLowerCase();
    filtered = filtered.filter((s) => s.secret_key.toLowerCase().includes(query));
  }

  // Apply filter mode
  switch (filter) {
    case "blocking":
      filtered = filtered.filter(
        (s) => !s.handling_strategy || s.handling_strategy === "none" || s.handling_strategy === ""
      );
      break;
    case "overridden":
      filtered = filtered.filter((s) =>
        overriddenSecrets.has(secretIdToString({ resource: s.resource_name, key: s.secret_key }))
      );
      break;
    case "excluded":
      filtered = filtered.filter((s) =>
        excludedSecrets.has(secretIdToString({ resource: s.resource_name, key: s.secret_key }))
      );
      break;
    case "all":
    default:
      break;
  }

  return filtered;
}

/**
 * Computes summary statistics from manifest and exclusion state
 */
export function computeManifestSummary(
  manifest: DeploymentManifestResponse,
  excludedResources: Set<string>,
  excludedSecrets: Set<string>,
  overriddenSecrets: Set<string>
): ManifestSummary {
  const secrets = manifest.secrets ?? [];
  let excludedCount = 0;
  let overriddenCount = 0;
  let blockingCount = 0;
  let strategizedCount = 0;

  for (const secret of secrets) {
    const secretId = secretIdToString({ resource: secret.resource_name, key: secret.secret_key });
    const isExcluded = excludedResources.has(secret.resource_name) || excludedSecrets.has(secretId);
    const isOverridden = overriddenSecrets.has(secretId);
    const hasStrategy = secret.handling_strategy && secret.handling_strategy !== "none" && secret.handling_strategy !== "";

    if (isExcluded) {
      excludedCount++;
    }
    if (isOverridden) {
      overriddenCount++;
    }
    if (!hasStrategy && !isExcluded) {
      blockingCount++;
    }
    if (hasStrategy) {
      strategizedCount++;
    }
  }

  return {
    totalSecrets: secrets.length,
    strategizedSecrets: strategizedCount,
    blockingSecrets: blockingCount,
    excludedSecrets: excludedCount,
    overriddenSecrets: overriddenCount,
    resourceCount: manifest.resources?.length ?? 0
  };
}

/**
 * Checks if a secret is considered "blocking" (no valid strategy)
 */
export function isSecretBlocking(secret: DeploymentManifestSecret): boolean {
  return !secret.handling_strategy || secret.handling_strategy === "none" || secret.handling_strategy === "";
}

/**
 * Gets the display label for a handling strategy
 */
export function getStrategyLabel(strategy: string): string {
  switch (strategy) {
    case "prompt":
      return "Prompt (ask user)";
    case "generate":
      return "Generate (auto)";
    case "strip":
      return "Strip (exclude)";
    case "delegate":
      return "Delegate (cloud)";
    case "none":
    case "":
      return "Not set";
    default:
      return strategy;
  }
}

/**
 * Gets the color class for a handling strategy
 */
export function getStrategyColorClass(strategy: string): string {
  switch (strategy) {
    case "prompt":
      return "text-emerald-200 border-emerald-400/30 bg-emerald-500/10";
    case "generate":
      return "text-cyan-200 border-cyan-400/30 bg-cyan-500/10";
    case "strip":
      return "text-amber-200 border-amber-400/30 bg-amber-500/10";
    case "delegate":
      return "text-purple-200 border-purple-400/30 bg-purple-500/10";
    default:
      return "text-white/60 border-white/10 bg-white/5";
  }
}

/**
 * Creates an export-ready manifest with exclusions applied
 */
export function createExportManifest(
  manifest: DeploymentManifestResponse,
  excludedResources: Set<string>,
  excludedSecrets: Set<string>
): DeploymentManifestResponse {
  const filteredSecrets = manifest.secrets.filter((secret) => {
    const secretId = secretIdToString({ resource: secret.resource_name, key: secret.secret_key });
    return !excludedResources.has(secret.resource_name) && !excludedSecrets.has(secretId);
  });

  const remainingResources = new Set(filteredSecrets.map((s) => s.resource_name));
  const filteredResourceNames = manifest.resources.filter(
    (r) => !excludedResources.has(r) && remainingResources.has(r)
  );

  // Recompute summary for filtered secrets
  const blockingSecrets = filteredSecrets.filter(isSecretBlocking);
  const strategizedSecrets = filteredSecrets.filter((s) => !isSecretBlocking(s));

  return {
    ...manifest,
    resources: filteredResourceNames,
    secrets: filteredSecrets,
    summary: {
      ...manifest.summary,
      total_secrets: filteredSecrets.length,
      strategized_secrets: strategizedSecrets.length,
      requires_action: blockingSecrets.length,
      blocking_secrets: blockingSecrets.map((s) => `${s.resource_name}/${s.secret_key}`)
    }
  };
}
