import { useState } from "react";
import {
  AlertCircle,
  Bot,
  Edit,
  Plus,
  RefreshCw,
  Settings2,
  Trash2,
  X,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { ScrollArea } from "../components/ui/scroll-area";
import { Textarea } from "../components/ui/textarea";
import { formatDate } from "../lib/utils";
import type { AgentProfile, CreateProfileRequest, RunnerType } from "../types";

interface ProfilesPageProps {
  profiles: AgentProfile[];
  loading: boolean;
  error: string | null;
  onCreateProfile: (profile: CreateProfileRequest) => Promise<AgentProfile>;
  onUpdateProfile: (id: string, profile: Partial<CreateProfileRequest>) => Promise<AgentProfile>;
  onDeleteProfile: (id: string) => Promise<void>;
  onRefresh: () => void;
}

const RUNNER_TYPES: RunnerType[] = ["claude-code", "codex", "opencode"];

export function ProfilesPage({
  profiles,
  loading,
  error,
  onCreateProfile,
  onUpdateProfile,
  onDeleteProfile,
  onRefresh,
}: ProfilesPageProps) {
  const [showForm, setShowForm] = useState(false);
  const [editingProfile, setEditingProfile] = useState<AgentProfile | null>(null);
  const [formData, setFormData] = useState<CreateProfileRequest>({
    name: "",
    description: "",
    runnerType: "claude-code",
    model: "claude-sonnet-4-20250514",
    maxTurns: 100,
    requiresSandbox: true,
    requiresApproval: true,
  });
  const [submitting, setSubmitting] = useState(false);

  const resetForm = () => {
    setFormData({
      name: "",
      description: "",
      runnerType: "claude-code",
      model: "claude-sonnet-4-20250514",
      maxTurns: 100,
      requiresSandbox: true,
      requiresApproval: true,
    });
    setEditingProfile(null);
    setShowForm(false);
  };

  const handleEdit = (profile: AgentProfile) => {
    setEditingProfile(profile);
    setFormData({
      name: profile.name,
      description: profile.description || "",
      runnerType: profile.runnerType,
      model: profile.model || "",
      maxTurns: profile.maxTurns || 100,
      requiresSandbox: profile.requiresSandbox,
      requiresApproval: profile.requiresApproval,
      allowedTools: profile.allowedTools,
      deniedTools: profile.deniedTools,
    });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      if (editingProfile) {
        await onUpdateProfile(editingProfile.id, formData);
      } else {
        await onCreateProfile(formData);
      }
      resetForm();
    } catch (err) {
      console.error("Failed to save profile:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this profile?")) return;
    try {
      await onDeleteProfile(id);
    } catch (err) {
      console.error("Failed to delete profile:", err);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Agent Profiles</h2>
          <p className="text-sm text-muted-foreground">
            Configure how agents execute tasks
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={onRefresh} className="gap-2">
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
          <Button size="sm" onClick={() => setShowForm(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            New Profile
          </Button>
        </div>
      </div>

      {error && (
        <Card className="border-destructive/50 bg-destructive/10">
          <CardContent className="flex items-center gap-3 py-4">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <p className="text-sm text-destructive">{error}</p>
          </CardContent>
        </Card>
      )}

      {/* Create/Edit Form */}
      {showForm && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>
                {editingProfile ? "Edit Profile" : "Create New Profile"}
              </CardTitle>
              <Button variant="ghost" size="icon" onClick={resetForm}>
                <X className="h-4 w-4" />
              </Button>
            </div>
            <CardDescription>
              {editingProfile
                ? "Update the agent profile configuration"
                : "Define how the agent should execute tasks"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
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
                  <Label htmlFor="runnerType">Runner Type *</Label>
                  <select
                    id="runnerType"
                    value={formData.runnerType}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        runnerType: e.target.value as RunnerType,
                      })
                    }
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    {RUNNER_TYPES.map((type) => (
                      <option key={type} value={type}>
                        {type}
                      </option>
                    ))}
                  </select>
                </div>
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

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="model">Model</Label>
                  <Input
                    id="model"
                    value={formData.model}
                    onChange={(e) =>
                      setFormData({ ...formData, model: e.target.value })
                    }
                    placeholder="e.g., claude-sonnet-4-20250514"
                  />
                </div>
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
              </div>

              <div className="flex gap-6">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.requiresSandbox}
                    onChange={(e) =>
                      setFormData({ ...formData, requiresSandbox: e.target.checked })
                    }
                    className="h-4 w-4 rounded border-input"
                  />
                  <span className="text-sm">Require Sandbox</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.requiresApproval}
                    onChange={(e) =>
                      setFormData({ ...formData, requiresApproval: e.target.checked })
                    }
                    className="h-4 w-4 rounded border-input"
                  />
                  <span className="text-sm">Require Approval</span>
                </label>
              </div>

              <div className="flex justify-end gap-2">
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
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      {/* Profiles List */}
      {loading ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            Loading profiles...
          </CardContent>
        </Card>
      ) : profiles.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <Bot className="h-16 w-16 mb-4 opacity-50" />
            <h3 className="text-lg font-semibold mb-1">No Agent Profiles</h3>
            <p className="text-sm mb-4">Create your first profile to get started</p>
            <Button onClick={() => setShowForm(true)} className="gap-2">
              <Plus className="h-4 w-4" />
              Create Profile
            </Button>
          </CardContent>
        </Card>
      ) : (
        <ScrollArea className="h-[600px]">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profiles.map((profile) => (
              <Card key={profile.id} className="relative">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-2">
                      <Settings2 className="h-5 w-5 text-primary" />
                      <CardTitle className="text-base">{profile.name}</CardTitle>
                    </div>
                    <div className="flex gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => handleEdit(profile)}
                      >
                        <Edit className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-destructive hover:text-destructive"
                        onClick={() => handleDelete(profile.id)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                  <CardDescription className="line-clamp-2">
                    {profile.description || "No description"}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex flex-wrap gap-2">
                    <Badge variant="secondary">{profile.runnerType}</Badge>
                    {profile.model && (
                      <Badge variant="outline" className="text-xs">
                        {profile.model}
                      </Badge>
                    )}
                  </div>
                  <div className="flex gap-2 text-xs">
                    {profile.requiresSandbox && (
                      <Badge variant="outline" className="text-[10px]">Sandbox</Badge>
                    )}
                    {profile.requiresApproval && (
                      <Badge variant="outline" className="text-[10px]">Approval</Badge>
                    )}
                    {profile.maxTurns && (
                      <Badge variant="outline" className="text-[10px]">
                        Max {profile.maxTurns} turns
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Created {formatDate(profile.createdAt)}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </ScrollArea>
      )}
    </div>
  );
}
