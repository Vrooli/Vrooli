/**
 * AutoSteerProfileEditorModal Component
 * Modal for creating and editing Auto Steer profiles with phases, tags, and conditions
 */

import { useState, useEffect } from 'react';
import { AlertCircle, Save, X } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { useCreateAutoSteerProfile, useUpdateAutoSteerProfile } from '@/hooks/useAutoSteer';
import type { AutoSteerProfile } from '@/types/api';
import { PhaseList } from './autosteer/PhaseList';
import { TagEditor } from './autosteer/TagEditor';
import { getApiErrorMessage, normalizeSteerMode } from '@/lib/utils';

interface AutoSteerProfileEditorModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  profile: AutoSteerProfile | null; // null = create new, non-null = edit existing
  prefillData?: Partial<AutoSteerProfile>; // For templates
}

interface SaveError {
  title: string;
  detail?: string;
  recovery?: string;
}

function normalizeProfileModes(source: Partial<AutoSteerProfile>): Partial<AutoSteerProfile> {
  const clone = JSON.parse(JSON.stringify(source)) as Partial<AutoSteerProfile>;
  if (clone.phases) {
    clone.phases = clone.phases.map((phase) => ({
      ...phase,
      mode: normalizeSteerMode(phase.mode),
    }));
  }
  return clone;
}

export function AutoSteerProfileEditorModal({
  open,
  onOpenChange,
  profile,
  prefillData,
}: AutoSteerProfileEditorModalProps) {
  const createProfile = useCreateAutoSteerProfile();
  const updateProfile = useUpdateAutoSteerProfile();

  const [localProfile, setLocalProfile] = useState<Partial<AutoSteerProfile>>({
    name: '',
    description: '',
    phases: [],
    tags: [],
  });
  const [saveError, setSaveError] = useState<SaveError | null>(null);

  // Initialize/reset local state when modal opens or profile changes
  useEffect(() => {
    if (open) {
      if (profile) {
        // Editing existing profile
        setLocalProfile(normalizeProfileModes(profile)); // Deep clone + normalize
      } else if (prefillData) {
        // Creating from template
        setLocalProfile(normalizeProfileModes(prefillData));
      } else {
        // Creating new profile
        setLocalProfile({
          name: '',
          description: '',
          phases: [],
          tags: [],
        });
      }
      setSaveError(null);
    }
  }, [open, profile, prefillData]);

  const handleSave = () => {
    setSaveError(null);
    const normalizedProfile = normalizeProfileModes(localProfile);

    // Validation
    if (!normalizedProfile.name?.trim()) {
      alert('Profile name is required');
      return;
    }

    if (!normalizedProfile.phases || normalizedProfile.phases.length === 0) {
      alert('At least one phase is required');
      return;
    }

    // Validate each phase
    for (const phase of normalizedProfile.phases) {
      if (!phase.mode) {
        alert('Each phase must have a mode');
        return;
      }
      if (!phase.max_iterations || phase.max_iterations < 1) {
        alert('Each phase must have max iterations >= 1');
        return;
      }
    }

    if (profile?.id) {
      // Update existing
      updateProfile.mutate(
        { id: profile.id, updates: normalizedProfile as AutoSteerProfile },
        {
          onSuccess: () => {
            onOpenChange(false);
          },
          onError: (error) => {
            setSaveError(buildSaveError(error));
          },
        }
      );
    } else {
      // Create new
      createProfile.mutate(normalizedProfile as AutoSteerProfile, {
        onSuccess: () => {
          onOpenChange(false);
        },
        onError: (error) => {
          setSaveError(buildSaveError(error));
        },
      });
    }
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  const updateField = (field: keyof AutoSteerProfile, value: any) => {
    setLocalProfile((prev) => ({ ...prev, [field]: value }));
    if (saveError) {
      setSaveError(null);
    }
  };

  const isLoading = createProfile.isPending || updateProfile.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {profile ? 'Edit Auto Steer Profile' : 'Create Auto Steer Profile'}
          </DialogTitle>
          <DialogDescription>
            Configure automated task guidance with phases, conditions, and tags
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Basic Info */}
          <div className="space-y-4">
            <div>
              <Label htmlFor="profile-name">Name *</Label>
              <Input
                id="profile-name"
                value={localProfile.name || ''}
                onChange={(e) => updateField('name', e.target.value)}
                placeholder="e.g., Progressive Enhancement"
                required
              />
            </div>

            <div>
              <Label htmlFor="profile-description">Description</Label>
              <Textarea
                id="profile-description"
                value={localProfile.description || ''}
                onChange={(e) => updateField('description', e.target.value)}
                placeholder="Optional description of this profile's purpose and behavior"
                rows={3}
              />
            </div>
          </div>

          {/* Tags */}
          <div>
            <Label>Tags</Label>
            <TagEditor
              tags={localProfile.tags || []}
              onChange={(tags) => updateField('tags', tags)}
            />
          </div>

          {/* Phases */}
          <div>
            <Label>Phases *</Label>
            <PhaseList
              phases={localProfile.phases || []}
              onChange={(phases) => updateField('phases', phases)}
            />
          </div>
        </div>

        {saveError && (
          <div className="rounded-md border border-red-500/40 bg-red-500/10 p-3 text-sm text-red-100">
            <div className="flex items-start gap-2">
              <AlertCircle className="h-4 w-4 mt-0.5 text-red-300" />
              <div className="space-y-1">
                <p className="font-semibold">{saveError.title}</p>
                {saveError.detail && (
                  <p className="text-xs text-red-200/90">{saveError.detail}</p>
                )}
                {saveError.recovery && (
                  <p className="text-xs text-red-200/80">{saveError.recovery}</p>
                )}
              </div>
            </div>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel} disabled={isLoading}>
            <X className="h-4 w-4 mr-2" />
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={isLoading}>
            <Save className="h-4 w-4 mr-2" />
            {isLoading ? 'Saving...' : 'Save Profile'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function buildSaveError(error: unknown): SaveError {
  const detail = getApiErrorMessage(error);
  const lowerDetail = detail.toLowerCase();

  if (lowerDetail.includes('invalid profile') || lowerDetail.includes('invalid mode')) {
    return {
      title: 'Profile validation failed',
      detail,
      recovery: 'Choose a valid phase mode and ensure each phase has at least one stop condition.',
    };
  }

  return {
    title: 'Unable to save profile',
    detail,
    recovery: 'Try again in a moment, or check the server logs if this keeps happening.',
  };
}
