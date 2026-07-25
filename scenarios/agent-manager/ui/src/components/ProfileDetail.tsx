import { Edit, Trash2 } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { networkAccessLabel, sandboxModeLabel } from "../lib/utils";
import { formatStandardDateTime } from "../lib/dateTime";
import type { AgentProfile } from "../types";
import { SandboxMode } from "../types";
import { durationMs, type Duration } from "@bufbuild/protobuf/wkt";

const durationToMinutes = (duration: Duration | undefined): number => {
  if (!duration) return 30;
  const ms = durationMs(duration);
  return Math.max(1, Math.round(ms / 60_000));
};

interface ProfileDetailProps {
  profile: AgentProfile;
  onEdit: (profile: AgentProfile) => void;
  onDelete: (profileId: string) => void;
}

export function ProfileDetail({ profile, onEdit, onDelete }: ProfileDetailProps) {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <h3 className="text-lg font-semibold">{profile.name}</h3>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => onEdit(profile)}
              className="gap-1"
            >
              <Edit className="h-4 w-4" />
              Edit
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onDelete(profile.id)}
              className="gap-1 text-destructive hover:text-destructive"
            >
              <Trash2 className="h-4 w-4" />
              Delete
            </Button>
          </div>
        </div>
        <p className="text-sm text-muted-foreground">
          {profile.description || "No description provided"}
        </p>
      </div>

      {/* Badges */}
      <div className="flex flex-wrap gap-2">
        <Badge variant="secondary">{profile.roleRef}</Badge>
        {profile.sandboxConfig?.mode != null && profile.sandboxConfig.mode !== SandboxMode.UNSPECIFIED && (
          <Badge variant="outline">Sandbox: {sandboxModeLabel(profile.sandboxConfig.mode)}</Badge>
        )}
        {profile.sandboxConfig?.manualReview && (
          <Badge variant="outline">Manual Review</Badge>
        )}
        {profile.networkAccess != null && (
          <Badge variant="outline">Net: {networkAccessLabel(profile.networkAccess)}</Badge>
        )}
        {profile.features?.enableBrowser && (
          <Badge variant="outline">Browser</Badge>
        )}
        {profile.extraFlags && Object.entries(profile.extraFlags).map(([runner, flagList]) => (
          flagList.flags?.map((flag, i) => (
            <Badge key={`${runner}-${i}`} variant="outline">{runner}: {flag}</Badge>
          ))
        ))}
      </div>

      {/* Configuration Details */}
      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-muted-foreground">Max Turns</span>
            <p className="font-medium">{profile.maxTurns || 100}</p>
          </div>
          <div>
            <span className="text-muted-foreground">Timeout</span>
            <p className="font-medium">{durationToMinutes(profile.timeout)} minutes</p>
          </div>
          {profile.effort && (
            <div>
              <span className="text-muted-foreground">Reasoning Effort</span>
              <p className="font-medium">{profile.effort}</p>
            </div>
          )}
        </div>

        {profile.profileKey && (
          <div className="text-sm">
            <span className="text-muted-foreground">Profile Key</span>
            <code className="block mt-1 text-xs bg-muted px-2 py-1 rounded">
              {profile.profileKey}
            </code>
          </div>
        )}

        <div className="grid grid-cols-2 gap-4 text-sm pt-2 border-t border-border">
          <div>
            <span className="text-muted-foreground">Created</span>
            <p className="font-medium">{formatStandardDateTime(profile.createdAt)}</p>
          </div>
          <div>
            <span className="text-muted-foreground">Updated</span>
            <p className="font-medium">{formatStandardDateTime(profile.updatedAt)}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
