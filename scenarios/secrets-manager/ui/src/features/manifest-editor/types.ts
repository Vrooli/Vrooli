import type { DeploymentManifestResponse, DeploymentManifestSecret } from "../../lib/api";

export type FilterMode = "all" | "blocking" | "overridden" | "excluded";

export interface ManifestEditorState {
  // Server data (source of truth)
  manifest: DeploymentManifestResponse;

  // Session-level exclusions (ephemeral, not saved)
  excludedResources: Set<string>;
  excludedSecrets: Set<string>; // Format: "resource:secret_key"

  // Pending overrides (not yet saved to server)
  pendingOverrides: Map<string, PendingOverrideEdit>;

  // UI state
  view: "tree" | "table";
  filter: FilterMode;
  searchQuery: string;
  expandedResources: Set<string>;
  selectedSecret: { resource: string; key: string } | null;
}

export interface PendingOverrideEdit {
  original: DeploymentManifestSecret;
  changes: Partial<OverrideFields>;
  isDirty: boolean;
}

export interface OverrideFields {
  handling_strategy: string;
  fallback_strategy?: string;
  requires_user_input: boolean;
  prompt_label?: string;
  prompt_description?: string;
  generator_template?: Record<string, unknown>;
  bundle_hints?: Record<string, unknown>;
}

export interface ResourceGroup {
  resourceName: string;
  secrets: DeploymentManifestSecret[];
  strategizedCount: number;
  totalCount: number;
  blockingCount: number;
  hasBlockers: boolean;
  isFullyExcluded: boolean;
}

export interface ManifestSummary {
  totalSecrets: number;
  strategizedSecrets: number;
  blockingSecrets: number;
  excludedSecrets: number;
  overriddenSecrets: number;
  resourceCount: number;
}

export interface SecretIdentifier {
  resource: string;
  key: string;
}

export const secretIdToString = (id: SecretIdentifier): string => `${id.resource}:${id.key}`;

export const stringToSecretId = (str: string): SecretIdentifier | null => {
  const colonIndex = str.indexOf(":");
  if (colonIndex === -1) return null;
  return {
    resource: str.slice(0, colonIndex),
    key: str.slice(colonIndex + 1)
  };
};
