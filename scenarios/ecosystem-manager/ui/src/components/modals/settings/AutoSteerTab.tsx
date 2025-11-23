/**
 * AutoSteerTab Component
 * Manage Auto Steer profiles for automated task guidance
 */

import { useState } from 'react';
import { Plus, Edit, Copy, Trash2, Zap } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useAutoSteerProfiles, useAutoSteerTemplates, useDeleteAutoSteerProfile } from '@/hooks/useAutoSteer';
import { AutoSteerProfileEditorModal } from '../AutoSteerProfileEditorModal';
import type { AutoSteerProfile, AutoSteerTemplate } from '@/types/api';

export function AutoSteerTab() {
  const { data: profiles = [], isLoading: loadingProfiles } = useAutoSteerProfiles();
  const { data: templates = [], isLoading: loadingTemplates } = useAutoSteerTemplates();
  const deleteProfile = useDeleteAutoSteerProfile();

  const [searchQuery, setSearchQuery] = useState('');
  const [isEditorOpen, setIsEditorOpen] = useState(false);
  const [editingProfile, setEditingProfile] = useState<AutoSteerProfile | null>(null);
  const [prefillData, setPrefillData] = useState<Partial<AutoSteerProfile> | undefined>(undefined);

  const filteredProfiles = profiles.filter((profile) =>
    profile.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (profile.description?.toLowerCase().includes(searchQuery.toLowerCase()) ?? false)
  );

  const handleDeleteProfile = (id: string, name: string) => {
    if (confirm(`Delete profile "${name}"?`)) {
      deleteProfile.mutate(id);
    }
  };

  const handleNewProfile = () => {
    setEditingProfile(null);
    setPrefillData(undefined);
    setIsEditorOpen(true);
  };

  const handleEditProfile = (profile: AutoSteerProfile) => {
    setEditingProfile(profile);
    setPrefillData(undefined);
    setIsEditorOpen(true);
  };

  const handleDuplicateProfile = (profile: AutoSteerProfile) => {
    setEditingProfile(null);
    setPrefillData({
      name: `${profile.name} (Copy)`,
      description: profile.description,
      phases: JSON.parse(JSON.stringify(profile.phases || [])),
      tags: [...(profile.tags || [])],
    });
    setIsEditorOpen(true);
  };

  const handleUseTemplate = (template: AutoSteerTemplate) => {
    setEditingProfile(null);
    setPrefillData({
      name: `${template.name} (Custom)`,
      description: template.description,
      phases: template.phases ? JSON.parse(JSON.stringify(template.phases)) : [],
      tags: [...(template.tags || [])],
    });
    setIsEditorOpen(true);
  };

  if (loadingProfiles) {
    return <div className="text-slate-400 text-center py-8">Loading profiles...</div>;
  }

  return (
    <div className="space-y-6">
      {/* Header & Search */}
      <div className="space-y-4">
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <Zap className="h-4 w-4" />
          <span>Create and manage Auto Steer profiles to guide task execution</span>
        </div>

        <div className="flex items-center gap-2">
          <Input
            type="search"
            placeholder="Search profiles..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="flex-1"
          />
          <Button size="sm" onClick={handleNewProfile}>
            <Plus className="h-4 w-4 mr-2" />
            New Profile
          </Button>
        </div>
      </div>

      {/* Profile List */}
      <div className="space-y-4">
        <h4 className="text-sm font-medium text-slate-300">
          Profiles ({filteredProfiles.length})
        </h4>

        {filteredProfiles.length === 0 ? (
          <div className="text-center py-12 text-slate-400">
            {searchQuery ? (
              <div>
                <p>No profiles match "{searchQuery}"</p>
                <Button variant="outline" size="sm" className="mt-4" onClick={() => setSearchQuery('')}>
                  Clear Search
                </Button>
              </div>
            ) : (
              <div>
                <p className="mb-2">No Auto Steer profiles yet</p>
                <p className="text-xs text-slate-500">
                  Create a profile to automate task guidance and decision-making
                </p>
              </div>
            )}
          </div>
        ) : (
          <div className="grid gap-3">
            {filteredProfiles.map((profile) => (
              <div
                key={profile.id}
                className="border border-white/10 rounded-lg p-4 hover:bg-slate-800/50 transition-colors"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <h5 className="text-sm font-medium text-slate-100 truncate">
                      {profile.name}
                    </h5>
                    {profile.description && (
                      <p className="text-xs text-slate-400 mt-1 line-clamp-2">
                        {profile.description}
                      </p>
                    )}
                    <div className="flex items-center gap-3 mt-2 text-xs text-slate-500">
                      {profile.phases && (
                        <span>{profile.phases.length} phase{profile.phases.length !== 1 ? 's' : ''}</span>
                      )}
                      {profile.tags && profile.tags.length > 0 && (
                        <span>• {profile.tags.length} tag{profile.tags.length !== 1 ? 's' : ''}</span>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-8 w-8 p-0"
                      title="Edit profile"
                      onClick={() => handleEditProfile(profile)}
                    >
                      <Edit className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-8 w-8 p-0"
                      title="Duplicate profile"
                      onClick={() => handleDuplicateProfile(profile)}
                    >
                      <Copy className="h-3.5 w-3.5" />
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-8 w-8 p-0 text-red-400 hover:text-red-300 border-red-400/30 hover:border-red-400/50"
                      title="Delete profile"
                      onClick={() => handleDeleteProfile(profile.id, profile.name)}
                      disabled={deleteProfile.isPending}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Template Gallery */}
      {!loadingTemplates && templates.length > 0 && (
        <div className="pt-6 border-t border-white/10 space-y-4">
          <h4 className="text-sm font-medium text-slate-300">
            Templates ({templates.length})
          </h4>

          <div className="grid gap-3">
            {templates.map((template) => (
              <div
                key={template.id}
                className="border border-white/10 rounded-lg p-4 bg-blue-900/10 hover:bg-blue-900/20 transition-colors"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <h5 className="text-sm font-medium text-blue-300 truncate">
                      {template.name}
                    </h5>
                    {template.description && (
                      <p className="text-xs text-slate-400 mt-1 line-clamp-2">
                        {template.description}
                      </p>
                    )}
                  </div>

                  <Button
                    variant="outline"
                    size="sm"
                    className="border-blue-400/30 hover:border-blue-400/50"
                    onClick={() => handleUseTemplate(template)}
                  >
                    <Plus className="h-3.5 w-3.5 mr-1.5" />
                    Use
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Info Section */}
      <div className="bg-blue-900/20 border border-blue-400/30 rounded-md p-4">
        <h4 className="text-sm font-medium text-blue-400 mb-2">About Auto Steer</h4>
        <ul className="text-xs text-slate-300 space-y-1">
          <li>• Auto Steer profiles guide task execution through multi-phase workflows</li>
          <li>• Each phase can have conditions, actions, and decision logic</li>
          <li>• Use templates as starting points or create custom profiles from scratch</li>
          <li>• Profiles can be shared across multiple tasks for consistency</li>
        </ul>
      </div>

      {/* Profile Editor Modal */}
      <AutoSteerProfileEditorModal
        open={isEditorOpen}
        onOpenChange={setIsEditorOpen}
        profile={editingProfile}
        prefillData={prefillData}
      />
    </div>
  );
}
