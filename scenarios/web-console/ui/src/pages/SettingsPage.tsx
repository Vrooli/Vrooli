// DOC: docs/concepts/ARCHITECTURE.md#file-map
// [REQ:P1-002a] Shortcut Profile Management UI
// [REQ:P1-003a] AI Provider Configuration UI
import { useState, useEffect, useCallback } from "react";
import { ArrowLeft, Plus, Trash2, Save, AlertCircle } from "lucide-react";
import { Button } from "../components/ui/button";
import ProviderHealthPanel from "../components/ProviderHealthPanel";
import {
  type ShortcutProfile,
  type ShortcutEntry,
  listShortcutProfiles,
  upsertShortcutProfile,
  deleteShortcutProfile,
  toErrorInfo,
} from "../lib/api";

interface SettingsPageProps {
  onBack: () => void;
}

function ShortcutEditor({
  profile,
  onSave,
  onDelete,
}: {
  profile: ShortcutProfile;
  onSave: (p: ShortcutProfile) => void;
  onDelete: (id: string) => void;
}) {
  const [entries, setEntries] = useState<ShortcutEntry[]>(profile.shortcuts);
  const [name, setName] = useState(profile.name);
  const [dirty, setDirty] = useState(false);

  const updateEntry = (idx: number, field: keyof ShortcutEntry, value: string) => {
    setEntries((prev) => prev.map((e, i) => (i === idx ? { ...e, [field]: value } : e)));
    setDirty(true);
  };

  const addEntry = () => {
    setEntries((prev) => [...prev, { label: "", command: "" }]);
    setDirty(true);
  };

  const removeEntry = (idx: number) => {
    setEntries((prev) => prev.filter((_, i) => i !== idx));
    setDirty(true);
  };

  return (
    <div data-testid={`shortcut-profile-${profile.id}`} className="rounded-lg border border-wc-default bg-wc-surface-input p-4">
      <div className="flex items-center justify-between mb-3">
        <input
          data-testid={`profile-name-${profile.id}`}
          className="bg-transparent text-sm font-medium text-wc-text-primary border-b border-transparent focus:border-wc-accent outline-none"
          value={name}
          onChange={(e) => { setName(e.target.value); setDirty(true); }}
        />
        <div className="flex gap-1">
          {dirty && (
            <Button
              data-testid={`profile-save-${profile.id}`}
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => {
                onSave({ ...profile, name, shortcuts: entries });
                setDirty(false);
              }}
              title="Save changes"
            >
              <Save className="h-3.5 w-3.5 text-green-400" />
            </Button>
          )}
          <Button
            data-testid={`profile-delete-${profile.id}`}
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => onDelete(profile.id)}
            title="Delete profile"
          >
            <Trash2 className="h-3.5 w-3.5 text-wc-text-faint hover:text-wc-error-detail" />
          </Button>
        </div>
      </div>

      <div className="space-y-2">
        {entries.map((entry, idx) => (
          <div key={idx} className="flex items-center gap-2">
            <input
              data-testid={`entry-label-${profile.id}-${idx}`}
              placeholder="Label"
              className="flex-1 rounded border border-wc-default bg-wc-surface-base px-2 py-1 text-xs text-wc-text-primary outline-none focus:border-wc-accent"
              value={entry.label}
              onChange={(e) => updateEntry(idx, "label", e.target.value)}
            />
            <input
              data-testid={`entry-command-${profile.id}-${idx}`}
              placeholder="Command"
              className="flex-[2] rounded border border-wc-default bg-wc-surface-base px-2 py-1 text-xs font-mono text-wc-text-primary outline-none focus:border-wc-accent"
              value={entry.command}
              onChange={(e) => updateEntry(idx, "command", e.target.value)}
            />
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 shrink-0"
              onClick={() => removeEntry(idx)}
            >
              <Trash2 className="h-3 w-3 text-wc-text-faint" />
            </Button>
          </div>
        ))}
      </div>

      <Button
        variant="ghost"
        size="sm"
        className="mt-2 text-xs text-wc-text-faint"
        onClick={addEntry}
      >
        <Plus className="mr-1 h-3 w-3" /> Add shortcut
      </Button>
    </div>
  );
}

export default function SettingsPage({ onBack }: SettingsPageProps) {
  const [profiles, setProfiles] = useState<ShortcutProfile[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const loadProfiles = useCallback(async (signal?: { cancelled: boolean }) => {
    setLoading(true);
    try {
      const data = await listShortcutProfiles();
      if (signal?.cancelled) return;
      setProfiles(data);
      setError(null);
    } catch (err) {
      if (signal?.cancelled) return;
      setError(toErrorInfo(err).message);
    } finally {
      if (!signal?.cancelled) setLoading(false);
    }
  }, []);

  useEffect(() => {
    const signal = { cancelled: false };
    loadProfiles(signal);
    return () => { signal.cancelled = true; };
  }, [loadProfiles]);

  const handleSave = useCallback(async (profile: ShortcutProfile) => {
    try {
      const updated = await upsertShortcutProfile({
        id: profile.id,
        scope: profile.scope,
        name: profile.name,
        shortcuts: profile.shortcuts,
      });
      setProfiles((prev) => prev.map((p) => (p.id === updated.id ? updated : p)));
    } catch (err) {
      setError(toErrorInfo(err).message);
      // Refetch to recover server-truth state on failure
      loadProfiles();
    }
  }, [loadProfiles]);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteShortcutProfile(id);
      setProfiles((prev) => prev.filter((p) => p.id !== id));
    } catch (err) {
      setError(toErrorInfo(err).message);
      // Refetch to recover server-truth state on failure
      loadProfiles();
    }
  }, [loadProfiles]);

  const handleCreate = useCallback(async () => {
    try {
      const newProfile = await upsertShortcutProfile({
        id: `profile-${Date.now()}`,
        scope: "workspace",
        name: "New Profile",
        shortcuts: [{ label: "List files", command: "ls -la" }],
      });
      setProfiles((prev) => [...prev, newProfile]);
    } catch (err) {
      setError(toErrorInfo(err).message);
    }
  }, []);

  return (
    <div className="flex h-screen flex-col bg-wc-surface-base text-wc-text-primary">
      <div className="flex items-center gap-3 border-b border-wc-default px-4 py-3">
        <Button
          data-testid="settings-back"
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={onBack}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-lg font-semibold">Settings</h1>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        {error && (
          <div data-testid="settings-error" className="flex items-center gap-2 rounded border border-wc-error bg-wc-error-bg px-3 py-2 text-sm text-wc-error-detail">
            <AlertCircle className="h-4 w-4 shrink-0" />
            {error}
          </div>
        )}

        {/* Shortcut Profiles Section */}
        <section>
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold uppercase tracking-wider text-wc-text-muted">
              Shortcut Profiles
            </h2>
            <Button
              data-testid="create-profile"
              variant="outline"
              size="sm"
              onClick={handleCreate}
            >
              <Plus className="mr-1 h-3 w-3" /> New Profile
            </Button>
          </div>

          {loading ? (
            <div className="text-center text-sm text-wc-text-faint py-4">Loading...</div>
          ) : profiles.length === 0 ? (
            <div className="text-center text-sm text-wc-text-faint py-4">
              No shortcut profiles configured
            </div>
          ) : (
            <div className="space-y-3">
              {profiles.map((profile) => (
                <ShortcutEditor
                  key={profile.id}
                  profile={profile}
                  onSave={handleSave}
                  onDelete={handleDelete}
                />
              ))}
            </div>
          )}
        </section>

        {/* AI Provider Config Section */}
        <section>
          <h2 className="text-sm font-semibold uppercase tracking-wider text-wc-text-muted mb-3">
            AI Providers
          </h2>
          <div className="rounded-lg border border-wc-default bg-wc-surface-input p-4">
            <ProviderHealthPanel open={true} />
          </div>
        </section>
      </div>
    </div>
  );
}
