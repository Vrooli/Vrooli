/**
 * URL state types for deployment routing.
 * Supports deep-linking to specific deployments, tabs, subtabs, and modals.
 */

/** Main tabs in deployment details view */
export type DeploymentTab =
  | "overview"
  | "live-state"
  | "files"
  | "drift"
  | "secrets"
  | "history"
  | "investigations"
  | "terminal";

/** Subtabs within the live-state tab */
export type LiveStateSubtab = "processes" | "ports" | "system" | "caddy" | "management";

/** Modal types that can be opened via URL */
export type ModalType = "redeploy" | "spawn-agent" | "delete" | "investigation-report";

/** Task types for spawn-agent modal */
export type SpawnAgentTaskType = "investigate" | "fix";

/** Complete URL state for deployment routing */
export interface DeploymentUrlState {
  /** Currently selected deployment ID, or null if on list view */
  deploymentId: string | null;
  /** Active main tab */
  tab: DeploymentTab;
  /** Active subtab within live-state tab */
  subtab: LiveStateSubtab;
  /** Currently open modal, or null if none */
  modal: ModalType | null;
  /** Additional modal parameters (e.g., taskType for spawn-agent, invId for investigation-report) */
  modalParams: Record<string, string>;
}

/** Valid tab values for validation */
export const VALID_TABS: readonly DeploymentTab[] = [
  "overview",
  "live-state",
  "files",
  "drift",
  "secrets",
  "history",
  "investigations",
  "terminal",
] as const;

/** Valid subtab values for validation */
export const VALID_SUBTABS: readonly LiveStateSubtab[] = [
  "processes",
  "ports",
  "system",
  "caddy",
  "management",
] as const;

/** Valid modal values for validation */
export const VALID_MODALS: readonly ModalType[] = [
  "redeploy",
  "spawn-agent",
  "delete",
  "investigation-report",
] as const;

/** Default URL state when no specific state is in URL */
export const DEFAULT_DEPLOYMENT_URL_STATE: DeploymentUrlState = {
  deploymentId: null,
  tab: "overview",
  subtab: "processes",
  modal: null,
  modalParams: {},
};
