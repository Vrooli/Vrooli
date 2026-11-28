export type HostMutation = "safeguard" | "tool" | "notification-recipient";
export type OnboardingTargetKind = "local" | "remote-node" | "unsupported";
export type RemoteHostNodeKind = "NODE_KIND_AGENT" | "NODE_KIND_SSH";

export interface TargetDescriptor {
  id: string;
  kind?: string;
}

export interface OnboardingTargetCapability {
  kind: OnboardingTargetKind;
  profileSession: "owner-scoped" | "unsupported";
  hostRequirements: "read-write" | "read-only" | "unsupported";
  hostPermissionWrites: boolean;
  reason?: string;
}

export interface HostMutationCapability {
  supported: boolean;
  reason?: string;
}

const LOCAL_ONLY_OPERATIONS: Record<HostMutation, string> = {
  safeguard: "Host safeguards can only be changed on the local host control plane",
  tool: "Host tool opt-ins can only be changed on the local host control plane",
  "notification-recipient": "Notification recipient can only be changed on the local operator-state owner",
};

/** Target-level capability contract shared by setup UI and write adapters.
 * A non-local identifier is a Bridge-routed node candidate; the owner read
 * establishes whether it is currently reachable before exposing local edits.
 */
export function onboardingTargetCapability(target: string, targets: TargetDescriptor[] = []): OnboardingTargetCapability {
  const selectedTarget = target.trim();
  if (selectedTarget === "local") return { kind: "local", profileSession: "owner-scoped", hostRequirements: "read-write", hostPermissionWrites: true };
  const remoteTarget = targets.find((candidate) => candidate.id === selectedTarget);
  if (remoteTarget && isRemoteHostNodeKind(remoteTarget.kind)) return { kind: "remote-node", profileSession: "owner-scoped", hostRequirements: "read-only", hostPermissionWrites: false, reason: "Remote host requirements are read-only here. Host tools and safeguards can only be changed by the local host control plane." };
  return { kind: "unsupported", profileSession: "unsupported", hostRequirements: "unsupported", hostPermissionWrites: false, reason: selectedTarget ? "This target is not a registered host supported by Onboarding." : "Choose the local host or an available remote node before editing host requirements." };
}

function isRemoteHostNodeKind(kind?: string): kind is RemoteHostNodeKind {
  return kind === "NODE_KIND_AGENT" || kind === "NODE_KIND_SSH";
}

/** UI and API-adapter guards for writes owned by the local control plane. */
export function hostMutationCapability(target: string, operation: HostMutation): HostMutationCapability {
  const selectedTarget = target.trim();
  if (selectedTarget === "local") return { supported: true };
  if (!selectedTarget) return { supported: false, reason: onboardingTargetCapability(target).reason };
  // This adapter only has authority to write through the local owner. It does
  // not have the target registry needed to classify an arbitrary ID as a
  // registered host, so report the actionable owner boundary for every
  // non-local write instead of guessing from an empty registry.
  return { supported: false, reason: LOCAL_ONLY_OPERATIONS[operation] };
}

export function requireHostMutationCapability(target: string, operation: HostMutation): void {
  const capability = hostMutationCapability(target, operation);
  if (!capability.supported) throw new Error(capability.reason);
}
