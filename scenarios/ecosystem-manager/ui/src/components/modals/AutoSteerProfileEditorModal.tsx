/**
 * AutoSteerProfileEditorModal Component
 * Modal for creating and editing Auto Steer profiles with phases, tags, and conditions
 */

import { useState, useEffect } from 'react';
import { Save, X } from 'lucide-react';
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

interface AutoSteerProfileEditorModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  profile: AutoSteerProfile | null; // null = create new, non-null = edit existing
  prefillData?: Partial<AutoSteerProfile>; // For templates
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

  // Initialize/reset local state when modal opens or profile changes
  useEffect(() => {
    if (open) {
      if (profile) {
        // Editing existing profile
        setLocalProfile(JSON.parse(JSON.stringify(profile))); // Deep clone
      } else if (prefillData) {
        // Creating from template
        setLocalProfile(JSON.parse(JSON.stringify(prefillData)));
      } else {
        // Creating new profile
        setLocalProfile({
          name: '',
          description: '',
          phases: [],
          tags: [],
        });
      }
    }
  }, [open, profile, prefillData]);

  const handleSave = () => {
    // Validation
    if (!localProfile.name?.trim()) {
      alert('Profile name is required');
      return;
    }

    if (!localProfile.phases || localProfile.phases.length === 0) {
      alert('At least one phase is required');
      return;
    }

    // Validate each phase
    for (const phase of localProfile.phases) {
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
        { id: profile.id, updates: localProfile as AutoSteerProfile },
        {
          onSuccess: () => {
            onOpenChange(false);
          },
        }
      );
    } else {
      // Create new
      createProfile.mutate(localProfile as AutoSteerProfile, {
        onSuccess: () => {
          onOpenChange(false);
        },
      });
    }
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  const updateField = (field: keyof AutoSteerProfile, value: any) => {
    setLocalProfile((prev) => ({ ...prev, [field]: value }));
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
