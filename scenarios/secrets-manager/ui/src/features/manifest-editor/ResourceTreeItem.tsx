import { ChevronDown, ChevronRight, AlertTriangle, CheckCircle2, Package } from "lucide-react";
import type { ResourceGroup } from "./types";
import type { DeploymentManifestSecret } from "../../lib/api";
import { SecretListItem } from "./SecretListItem";

interface ResourceTreeItemProps {
  group: ResourceGroup;
  isExpanded: boolean;
  selectedSecretId: { resource: string; key: string } | null;
  expandedResources: Set<string>;
  excludedSecrets: Set<string>;
  overriddenSecrets: Set<string>;
  onToggleExpand: () => void;
  onToggleResourceExclusion: () => void;
  onSelectSecret: (resource: string, key: string) => void;
  onToggleSecretExclusion: (resource: string, key: string) => void;
}

export function ResourceTreeItem({
  group,
  isExpanded,
  selectedSecretId,
  excludedSecrets,
  overriddenSecrets,
  onToggleExpand,
  onToggleResourceExclusion,
  onSelectSecret,
  onToggleSecretExclusion
}: ResourceTreeItemProps) {
  const isSecretExcluded = (secret: DeploymentManifestSecret) =>
    group.isFullyExcluded || excludedSecrets.has(`${secret.resource_name}:${secret.secret_key}`);

  const isSecretOverridden = (secret: DeploymentManifestSecret) =>
    overriddenSecrets.has(`${secret.resource_name}:${secret.secret_key}`);

  return (
    <div className="border-b border-white/5 last:border-b-0">
      <div
        className={`flex items-center gap-2 px-2 py-2 cursor-pointer transition-colors hover:bg-white/5 ${
          group.isFullyExcluded ? "opacity-50" : ""
        }`}
      >
        <input
          type="checkbox"
          checked={!group.isFullyExcluded}
          onChange={(e) => {
            e.stopPropagation();
            onToggleResourceExclusion();
          }}
          className="h-4 w-4 rounded border-white/20 bg-slate-800 text-emerald-500 focus:ring-emerald-500/50 cursor-pointer"
          title={group.isFullyExcluded ? "Include resource" : "Exclude resource"}
        />
        <button
          type="button"
          onClick={onToggleExpand}
          className="flex flex-1 items-center gap-2 text-left min-w-0"
        >
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-white/60 shrink-0" />
          ) : (
            <ChevronRight className="h-4 w-4 text-white/60 shrink-0" />
          )}
          <Package className="h-4 w-4 text-white/60 shrink-0" />
          <span
            className={`text-sm font-medium text-white truncate ${
              group.isFullyExcluded ? "line-through" : ""
            }`}
          >
            {group.resourceName}
          </span>
          <span className="text-xs text-white/50">
            ({group.strategizedCount}/{group.totalCount})
          </span>
          {group.hasBlockers ? (
            <span title="Has blocking secrets" className="ml-auto">
              <AlertTriangle className="h-4 w-4 text-amber-400 shrink-0" />
            </span>
          ) : (
            <span title="All strategized" className="ml-auto">
              <CheckCircle2 className="h-4 w-4 text-emerald-400 shrink-0" />
            </span>
          )}
        </button>
      </div>

      {isExpanded && (
        <div className="pl-6 pb-1">
          {group.secrets.map((secret) => (
            <SecretListItem
              key={`${secret.resource_name}:${secret.secret_key}`}
              secret={secret}
              isSelected={
                selectedSecretId?.resource === secret.resource_name &&
                selectedSecretId?.key === secret.secret_key
              }
              isExcluded={isSecretExcluded(secret)}
              isOverridden={isSecretOverridden(secret)}
              onSelect={() => onSelectSecret(secret.resource_name, secret.secret_key)}
              onToggleExclude={() => onToggleSecretExclusion(secret.resource_name, secret.secret_key)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
