import type { ResourceGroup } from "./types";
import { ResourceTreeItem } from "./ResourceTreeItem";

interface ResourceTreeProps {
  groups: ResourceGroup[];
  expandedResources: Set<string>;
  selectedSecretId: { resource: string; key: string } | null;
  excludedResources: Set<string>;
  excludedSecrets: Set<string>;
  overriddenSecrets: Set<string>;
  onToggleResource: (resourceName: string) => void;
  onToggleResourceExclusion: (resourceName: string) => void;
  onSelectSecret: (resource: string, key: string) => void;
  onToggleSecretExclusion: (resource: string, key: string) => void;
}

export function ResourceTree({
  groups,
  expandedResources,
  selectedSecretId,
  excludedResources,
  excludedSecrets,
  overriddenSecrets,
  onToggleResource,
  onToggleResourceExclusion,
  onSelectSecret,
  onToggleSecretExclusion
}: ResourceTreeProps) {
  if (groups.length === 0) {
    return (
      <div className="px-4 py-8 text-center text-sm text-white/50">
        No resources match the current filter.
      </div>
    );
  }

  return (
    <div className="divide-y divide-white/5">
      {groups.map((group) => (
        <ResourceTreeItem
          key={group.resourceName}
          group={group}
          isExpanded={expandedResources.has(group.resourceName)}
          selectedSecretId={selectedSecretId}
          expandedResources={expandedResources}
          excludedSecrets={excludedSecrets}
          overriddenSecrets={overriddenSecrets}
          onToggleExpand={() => onToggleResource(group.resourceName)}
          onToggleResourceExclusion={() => onToggleResourceExclusion(group.resourceName)}
          onSelectSecret={onSelectSecret}
          onToggleSecretExclusion={onToggleSecretExclusion}
        />
      ))}
    </div>
  );
}
