import { Edit, Trash2 } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { formatDate, runnerTypeLabel } from "../lib/utils";
import type { AgentProfile } from "../types";
import { ModelPreset } from "../types";
import { durationMs, type Duration } from "@bufbuild/protobuf/wkt";

const durationToMinutes = (duration: Duration | undefined): number => {
  if (!duration) return 30;
  const ms = durationMs(duration);
  return Math.max(1, Math.round(ms / 60_000));
};

const modelPresetLabel = (preset?: ModelPreset) => {
  switch (preset) {
    case ModelPreset.FAST:
      return "Fast";
    case ModelPreset.CHEAP:
      return "Cheap";
    case ModelPreset.SMART:
      return "Smart";
    default:
      return "";
  }
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
        <Badge variant="secondary">{runnerTypeLabel(profile.runnerType)}</Badge>
        {profile.modelPreset !== ModelPreset.UNSPECIFIED && (
          <Badge variant="outline">Preset: {modelPresetLabel(profile.modelPreset)}</Badge>
        )}
        {profile.model && profile.model.trim() !== "" && (
          <Badge variant="outline">{profile.model}</Badge>
        )}
        {profile.requiresSandbox && (
          <Badge variant="outline">Sandbox Required</Badge>
        )}
        {profile.requiresApproval && (
          <Badge variant="outline">Approval Required</Badge>
        )}
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
        </div>

        {profile.profileKey && (
          <div className="text-sm">
            <span className="text-muted-foreground">Profile Key</span>
            <code className="block mt-1 text-xs bg-muted px-2 py-1 rounded">
              {profile.profileKey}
            </code>
          </div>
        )}

        {profile.fallbackRunnerTypes && profile.fallbackRunnerTypes.length > 0 && (
          <div className="text-sm">
            <span className="text-muted-foreground">Fallback Runners</span>
            <div className="flex flex-wrap gap-1 mt-1">
              {profile.fallbackRunnerTypes.map((rt, i) => (
                <Badge key={i} variant="outline" className="text-xs">
                  {runnerTypeLabel(rt)}
                </Badge>
              ))}
            </div>
          </div>
        )}

        <div className="grid grid-cols-2 gap-4 text-sm pt-2 border-t border-border">
          <div>
            <span className="text-muted-foreground">Created</span>
            <p className="font-medium">{formatDate(profile.createdAt)}</p>
          </div>
          <div>
            <span className="text-muted-foreground">Updated</span>
            <p className="font-medium">{formatDate(profile.updatedAt)}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
