import { Zap } from 'lucide-react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { formatSkillSetLabel, formatSkillSetTooltip } from '@/lib/utils';
import { useMergedSkillNames } from '@/hooks/usePromptFiles';
import type { AutoSteerProfile } from '@/types/api';

interface ProfilePanelProps {
  value?: string;
  onChange: (profileId: string | undefined) => void;
  profiles: AutoSteerProfile[];
  isLoading?: boolean;
}

export function ProfilePanel({ value, onChange, profiles, isLoading }: ProfilePanelProps) {
  const selectedProfile = profiles.find((p) => p.id === value);
  const { data: skillNames = [] } = useMergedSkillNames();

  const weightedDimensions = Object.entries(selectedProfile?.objective?.dimension_weights ?? {})
    .sort(([, a], [, b]) => b - a);
  const allowedSkills = selectedProfile?.allowed_skills ?? [];

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3 mb-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-indigo-500/10 shrink-0">
          <Zap className="h-5 w-5 text-indigo-400" />
        </div>
        <div>
          <h3 className="text-sm font-medium text-slate-200">Auto Steer Profile</h3>
          <p className="text-sm text-slate-400 mt-0.5">
            Run the closed-loop controller against an objective — it picks skills to close the
            heaviest open dimensions until the objective is met or the budget is spent.
          </p>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="profile-select">Profile</Label>
        <Select
          value={value || ''}
          onValueChange={(val) => onChange(val || undefined)}
          disabled={isLoading}
        >
          <SelectTrigger id="profile-select" className="w-full">
            <SelectValue placeholder={isLoading ? 'Loading profiles...' : 'Select a profile'} />
          </SelectTrigger>
          <SelectContent>
            {profiles.length === 0 ? (
              <div className="px-2 py-1.5 text-sm text-slate-400">No profiles available</div>
            ) : (
              profiles.map((profile) => (
                <SelectItem key={profile.id} value={profile.id}>
                  {profile.name}
                </SelectItem>
              ))
            )}
          </SelectContent>
        </Select>
      </div>

      {selectedProfile && (
        <div className="rounded-md border border-indigo-500/20 bg-indigo-500/5 p-3 space-y-2">
          {selectedProfile.description && (
            <p className="text-sm text-slate-400">{selectedProfile.description}</p>
          )}
          <div className="flex items-center gap-2 text-xs text-slate-500">
            <span className="font-medium text-indigo-400">
              {allowedSkills.length} skill{allowedSkills.length === 1 ? '' : 's'}
            </span>
            {weightedDimensions.length > 0 && (
              <>
                <span className="text-slate-600">|</span>
                <span>
                  {weightedDimensions.length} weighted dimension{weightedDimensions.length === 1 ? '' : 's'}
                </span>
              </>
            )}
            {selectedProfile.tags && selectedProfile.tags.length > 0 && (
              <>
                <span className="text-slate-600">|</span>
                <span>{selectedProfile.tags.join(', ')}</span>
              </>
            )}
          </div>
          {weightedDimensions.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mt-2">
              {weightedDimensions.map(([dim, weight]) => (
                <span
                  key={dim}
                  className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-indigo-500/10 text-indigo-300 border border-indigo-500/20"
                  title={`Weight ${weight}`}
                >
                  {dim} ×{weight}
                </span>
              ))}
            </div>
          )}
          {allowedSkills.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mt-2">
              {allowedSkills.map((skill) => (
                <span
                  key={skill}
                  className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-slate-700/40 text-slate-300 border border-white/10"
                  title={formatSkillSetTooltip([skill], skillNames)}
                >
                  {formatSkillSetLabel([skill], skillNames, { maxVisible: 1, emptyLabel: skill })}
                </span>
              ))}
            </div>
          )}
        </div>
      )}

      {profiles.length === 0 && !isLoading && (
        <p className="text-xs text-slate-500">
          No profiles found. Create profiles in Settings &gt; Auto Steer.
        </p>
      )}
    </div>
  );
}
