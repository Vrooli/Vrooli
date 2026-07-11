import { memo, useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  AlertCircle,
  Bot,
  Plus,
  RefreshCw,
  Settings2,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent } from "../components/ui/card";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { RoleSelector } from "../components/RoleSelector";
import { Textarea } from "../components/ui/textarea";
import { durationMs, type Duration } from "@bufbuild/protobuf/wkt";
import { profileSandboxModeFormValue } from "../lib/utils";
import type { RolePolicyCatalog } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import type { AgentProfile, ProfileFormData } from "../types";
import { NetworkAccess } from "../types";
import { ProfileDetail } from "../components/ProfileDetail";
import { useViewportSize } from "../hooks/useViewportSize";
import { formatStandardDateTime } from "../lib/dateTime";

import { MasterDetailLayout, ListPanel, DetailPanel } from "../components/patterns/MasterDetail";
import { SearchToolbar, type FilterConfig, type SortOption } from "../components/patterns/SearchToolbar";
import { BoundedList, ListItem, ListItemTitle, ListItemSubtitle } from "../components/patterns/ListItem";

interface ProfilesPageProps {
  profiles: AgentProfile[];
  loading: boolean;
  error: string | null;
  onCreateProfile: (profile: ProfileFormData) => Promise<AgentProfile>;
  onUpdateProfile: (id: string, profile: ProfileFormData) => Promise<AgentProfile>;
  onDeleteProfile: (id: string) => Promise<void>;
  onRefresh: () => void;
  rolePolicyCatalog?: RolePolicyCatalog;
}

const SORT_OPTIONS: SortOption[] = [
  { value: "newest", label: "Newest First" },
  { value: "oldest", label: "Oldest First" },
  { value: "name", label: "Name A-Z" },
];

const durationToMinutes = (duration: Duration | undefined): number => {
  if (!duration) return 30;
  const ms = durationMs(duration);
  return Math.max(1, Math.round(ms / 60_000));
};

type ProfileFormState = ProfileFormData;

interface ProfileListRowProps {
  profile: AgentProfile;
  selected: boolean;
  onSelect: (profileId: string) => void;
}

const ProfileListRow = memo(function ProfileListRow({
  profile,
  selected,
  onSelect,
}: ProfileListRowProps) {
  return (
    <ListItem
      selected={selected}
      onClick={() => onSelect(profile.id)}
      icon={<Settings2 className="h-5 w-5 text-primary flex-shrink-0" />}
      actions={<Badge variant="secondary">{profile.roleRef}</Badge>}
    >
      <ListItemTitle>{profile.name}</ListItemTitle>
      <ListItemSubtitle>
        {profile.description || "No description"} | {formatStandardDateTime(profile.createdAt)}
      </ListItemSubtitle>
    </ListItem>
  );
});

export function ProfilesPage({
  profiles,
  loading,
  error,
  onCreateProfile,
  onUpdateProfile,
  onDeleteProfile,
  onRefresh,
  rolePolicyCatalog,
}: ProfilesPageProps) {
  const { isDesktop } = useViewportSize();

  // Selection state
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [searchParams] = useSearchParams();
  const profileIdParam = searchParams.get("profileId");
  const profileKeyParam = searchParams.get("profileKey");

  // Modal state
  const [showForm, setShowForm] = useState(false);
  const [editingProfile, setEditingProfile] = useState<AgentProfile | null>(null);
  const [formData, setFormData] = useState<ProfileFormState>({
    name: "",
    profileKey: "",
    description: "",
    roleRef: rolePolicyCatalog?.defaultRole || "code.default",
    maxTurns: 100,
    sandboxMode: "protected" as const,
    networkAccess: "localhost" as const,
    timeoutMinutes: 30,
    features: { enableBrowser: false },
    extraFlags: {},
  });
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // Filter/sort/search state
  const [searchQuery, setSearchQuery] = useState("");
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [sortBy, setSortBy] = useState<string>("newest");

  const selectedProfile = useMemo(
    () => profiles.find((p) => p.id === selectedProfileId) || null,
    [profiles, selectedProfileId]
  );

  // Handle deep-linking via profileId or profileKey query params
  useEffect(() => {
    if (profileIdParam) {
      setSelectedProfileId(profileIdParam);
    } else if (profileKeyParam && profiles.length > 0) {
      // Find profile by key
      const profileByKey = profiles.find((p) => p.profileKey === profileKeyParam);
      if (profileByKey) {
        setSelectedProfileId(profileByKey.id);
      }
    }
  }, [profileIdParam, profileKeyParam, profiles]);

  const resetForm = () => {
    setFormData({
      name: "",
      profileKey: "",
      description: "",
      roleRef: rolePolicyCatalog?.defaultRole || "code.default",
      maxTurns: 100,
      sandboxMode: "protected",
      networkAccess: "localhost" as const,
      timeoutMinutes: 30,
      features: { enableBrowser: false },
      extraFlags: {},
    });
    setEditingProfile(null);
    setShowForm(false);
    setFormError(null);
  };

  const handleEdit = (profile: AgentProfile) => {
    setEditingProfile(profile);
    setFormData({
      name: profile.name,
      profileKey: profile.profileKey || "",
      description: profile.description || "",
      roleRef: profile.roleRef,
      maxTurns: profile.maxTurns || 100,
      sandboxMode: profileSandboxModeFormValue(profile),
      networkAccess: profile.networkAccess === NetworkAccess.NONE ? "none"
        : profile.networkAccess === NetworkAccess.FULL ? "full"
        : "localhost",
      allowedTools: profile.allowedTools,
      deniedTools: profile.deniedTools,
      timeoutMinutes: durationToMinutes(profile.timeout),
      features: {
        enableBrowser: profile.features?.enableBrowser ?? false,
      },
      extraFlags: Object.fromEntries(
        Object.entries(profile.extraFlags ?? {}).map(([rt, list]) => [rt, list.flags ?? []])
      ),
    });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);
    try {
      const normalizedProfile: ProfileFormData = {
        ...formData,
        roleRef: formData.roleRef.trim(),
        timeoutMinutes: formData.timeoutMinutes ?? 30,
      };
      if (editingProfile) {
        await onUpdateProfile(editingProfile.id, normalizedProfile);
      } else {
        await onCreateProfile(normalizedProfile);
      }
      resetForm();
    } catch (err) {
      setFormError((err as Error).message);
      console.error("Failed to save profile:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this profile?")) return;
    try {
      await onDeleteProfile(id);
      if (selectedProfileId === id) {
        setSelectedProfileId(null);
      }
    } catch (err) {
      console.error("Failed to delete profile:", err);
    }
  };

  const filteredAndSortedProfiles = useMemo(() => {
    let result = [...profiles];

    if (roleFilter !== "all") {
      result = result.filter((p) => p.roleRef === roleFilter);
    }

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (p) =>
          p.name.toLowerCase().includes(query) ||
          p.description?.toLowerCase().includes(query)
      );
    }

    result.sort((a, b) => {
      if (sortBy === "name") {
        return a.name.localeCompare(b.name);
      }
      const aTime = a.createdAt ? new Date(a.createdAt.toString()).getTime() : 0;
      const bTime = b.createdAt ? new Date(b.createdAt.toString()).getTime() : 0;
      return sortBy === "newest" ? bTime - aTime : aTime - bTime;
    });

    return result;
  }, [profiles, roleFilter, searchQuery, sortBy]);

  useEffect(() => {
    if (!isDesktop) return;
    if (filteredAndSortedProfiles.length === 0) return;

    const hasSelection =
      selectedProfileId !== null &&
      filteredAndSortedProfiles.some((profile) => profile.id === selectedProfileId);

    if (!hasSelection) {
      const first = filteredAndSortedProfiles[0];
      if (first) setSelectedProfileId(first.id);
    }
  }, [filteredAndSortedProfiles, isDesktop, selectedProfileId]);

  const getProfileKey = useCallback((profile: AgentProfile) => profile.id, []);
  const handleSelectProfile = useCallback((profileId: string) => {
    setSelectedProfileId(profileId);
  }, []);
  const renderProfileRow = useCallback(
    (profile: AgentProfile) => (
      <ProfileListRow
        profile={profile}
        selected={selectedProfileId === profile.id}
        onSelect={handleSelectProfile}
      />
    ),
    [handleSelectProfile, selectedProfileId]
  );

  const filters: FilterConfig[] = [
    {
      id: "roleRef",
      label: "Filter by role",
      value: roleFilter,
      options: (rolePolicyCatalog?.roles ?? []).map((role) => ({ value: role.roleRef, label: role.description || role.roleRef })),
      onChange: setRoleFilter,
      allLabel: "All Roles",
    },
  ];

  const listPanel = (
    <ListPanel
      title="Agent Profiles"
      count={filteredAndSortedProfiles.length}
      loading={loading}
      headerActions={
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={onRefresh}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Button size="sm" onClick={() => setShowForm(true)} className="gap-1">
            <Plus className="h-4 w-4" />
            <span className="hidden sm:inline">New</span>
          </Button>
        </div>
      }
      toolbar={
        <SearchToolbar
          searchValue={searchQuery}
          onSearchChange={setSearchQuery}
          searchPlaceholder="Search profiles..."
          filters={filters}
          sortOptions={SORT_OPTIONS}
          currentSort={sortBy}
          onSortChange={setSortBy}
        />
      }
      empty={
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Bot className="h-12 w-12 mb-3 opacity-50" />
          <p className="font-medium">
            {profiles.length === 0 ? "No Agent Profiles" : "No Matching Profiles"}
          </p>
          <p className="text-sm text-center mt-1">
            {profiles.length === 0
              ? "Create your first profile to get started"
              : "Try adjusting your filters"}
          </p>
          {profiles.length === 0 && (
            <Button
              onClick={() => setShowForm(true)}
              className="gap-2 mt-4"
              size="sm"
            >
              <Plus className="h-4 w-4" />
              Create Profile
            </Button>
          )}
        </div>
      }
    >
      <BoundedList
        items={filteredAndSortedProfiles}
        getKey={getProfileKey}
        renderItem={renderProfileRow}
      />
    </ListPanel>
  );

  const detailPanel = (
    <DetailPanel
      title="Profile Details"
      hasSelection={!!selectedProfile}
      empty={
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Settings2 className="h-12 w-12 mb-3 opacity-50" />
          <p className="text-sm">Select a profile to view details</p>
        </div>
      }
    >
      {selectedProfile && (
        <ProfileDetail
          profile={selectedProfile}
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
      )}
    </DetailPanel>
  );

  // Build header content with error banner
  const headerContent = error ? (
    <Card className="border-destructive/50 bg-destructive/10">
      <CardContent className="flex items-center gap-3 py-4">
        <AlertCircle className="h-4 w-4 text-destructive" />
        <p className="text-sm text-destructive">{error}</p>
      </CardContent>
    </Card>
  ) : null;

  return (
    <>
      <MasterDetailLayout
        storageKey="profiles"
        headerContent={headerContent}
        listPanel={listPanel}
        detailPanel={detailPanel}
        selectedId={selectedProfileId}
        onDeselect={() => setSelectedProfileId(null)}
        detailTitle={selectedProfile?.name ?? "Profile Details"}
      />

      {/* Create/Edit Profile Modal */}
      <Dialog open={showForm} onOpenChange={(open) => !open && resetForm()} fullScreenMobile>
        <DialogContent fullScreenMobile>
          <DialogHeader onClose={resetForm}>
            <DialogTitle>
              {editingProfile ? "Edit Profile" : "Create New Profile"}
            </DialogTitle>
            <DialogDescription>
              {editingProfile
                ? "Update the agent profile configuration"
                : "Define how the agent should execute tasks"}
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="flex flex-col flex-1 min-h-0 overflow-hidden">
            <DialogBody className="space-y-4">
              {formError && (
                <Card className="border-destructive/50 bg-destructive/10">
                  <CardContent className="flex items-center gap-3 py-3">
                    <AlertCircle className="h-4 w-4 text-destructive" />
                    <p className="text-sm text-destructive">{formError}</p>
                  </CardContent>
                </Card>
              )}
              <div className="space-y-2">
                  <Label htmlFor="name">Name *</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) =>
                      setFormData({ ...formData, name: e.target.value })
                    }
                    placeholder="e.g., Claude Code Default"
                    required
                  />
              </div>

              <div className="space-y-2">
                <Label htmlFor="profileKey">Profile Key</Label>
                <Input
                  id="profileKey"
                  value={formData.profileKey ?? ""}
                  onChange={(e) =>
                    setFormData({ ...formData, profileKey: e.target.value })
                  }
                  placeholder="auto-generated from name if left blank"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={formData.description}
                  onChange={(e) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  placeholder="Describe what this profile is for..."
                  rows={2}
                />
              </div>

              <RoleSelector
                catalog={rolePolicyCatalog}
                value={formData.roleRef}
                onChange={(roleRef) => setFormData({ ...formData, roleRef })}
                label="Execution Role"
              />

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="maxTurns">Max Turns</Label>
                  <Input
                    id="maxTurns"
                    type="number"
                    value={formData.maxTurns}
                    onChange={(e) =>
                      setFormData({ ...formData, maxTurns: parseInt(e.target.value) || 100 })
                    }
                    min={1}
                    max={1000}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="timeout">Timeout (minutes)</Label>
                  <Input
                    id="timeout"
                    type="number"
                    value={formData.timeoutMinutes ?? 30}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        timeoutMinutes: parseInt(e.target.value) || 30,
                      })
                    }
                    min={1}
                    max={1440}
                  />
                </div>
              </div>

              <div className="flex gap-6">
                <label className="flex items-center gap-2">
                  <span className="text-sm">Sandbox Mode</span>
                  <select
                    value={formData.sandboxMode ?? "protected"}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        sandboxMode: e.target.value as "off" | "tracking" | "protected",
                      })
                    }
                    className="h-9 rounded-md border border-input bg-background px-2 text-sm"
                  >
                    <option value="off">Off</option>
                    <option value="tracking">Tracking</option>
                    <option value="protected">Protected</option>
                  </select>
                </label>
                {/* The "Require Approval" toggle was removed in
                    agent-sandbox-audit-foundation Phase 3b — operator-gated
                    apply lives on SandboxConfig.manualReview now. The
                    "Require Sandbox" boolean was removed in agent-manager
                    Phase 1: SandboxConfig.mode is the single source of
                    truth (see DeriveRunMode in domain/decisions.go). */}
                <label className="flex items-center gap-2">
                  <span className="text-sm">Network Access</span>
                  <select
                    value={formData.networkAccess ?? "localhost"}
                    onChange={(e) =>
                      setFormData({ ...formData, networkAccess: e.target.value as "none" | "localhost" | "full" })
                    }
                    className="h-8 rounded border border-input bg-background px-2 text-sm"
                  >
                    <option value="none">None</option>
                    <option value="localhost">Localhost</option>
                    <option value="full">Full</option>
                  </select>
                </label>
              </div>

              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.features?.enableBrowser ?? false}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      features: { ...formData.features, enableBrowser: e.target.checked },
                    })
                  }
                  className="h-4 w-4 rounded border-input"
                />
                <span className="text-sm">Request browser automation when the resolved runner supports it</span>
              </label>
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={resetForm}>
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting
                  ? "Saving..."
                  : editingProfile
                  ? "Update Profile"
                  : "Create Profile"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
