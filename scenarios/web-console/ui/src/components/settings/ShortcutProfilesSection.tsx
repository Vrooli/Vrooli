import { useCallback, useEffect, useState } from "react";
import { AlertCircle, Plus, Save, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import { SettingsCard, SettingsSectionIntro } from "./primitives";
import type { Profile as ShortcutProfile } from "@vrooli/proto-types/web-console/v1/shortcuts/shortcuts_pb";
import { shortcutsClient } from "../../api/shortcuts";
import { strings } from "../../consts/strings";
import { toErrorInfo } from "../../lib/errors";

interface ShortcutDraft {
  label: string;
  command: string;
  description: string;
}

interface ProfileDraft {
  id: string;
  scope: string;
  name: string;
  shortcuts: ShortcutDraft[];
}

function ShortcutEditor({
  profile,
  onSave,
  onDelete,
}: {
  profile: ShortcutProfile;
  onSave: (draft: ProfileDraft) => void;
  onDelete: (id: string) => void;
}) {
  const { t } = useTranslation();
  const [entries, setEntries] = useState<ShortcutDraft[]>(
    profile.shortcuts.map((s) => ({ label: s.label, command: s.command, description: s.description })),
  );
  const [name, setName] = useState(profile.name);
  const [dirty, setDirty] = useState(false);

  const updateEntry = (index: number, field: keyof ShortcutDraft, value: string) => {
    setEntries((current) => current.map((entry, idx) => (
      idx === index ? { ...entry, [field]: value } : entry
    )));
    setDirty(true);
  };

  const addEntry = () => {
    setEntries((current) => [...current, { label: "", command: "", description: "" }]);
    setDirty(true);
  };

  const removeEntry = (index: number) => {
    setEntries((current) => current.filter((_, idx) => idx !== index));
    setDirty(true);
  };

  return (
    <div
      data-testid={`shortcut-profile-${profile.id}`}
      className="rounded-xl border border-wc-default bg-wc-surface-base/70 p-3"
    >
      <div className="mb-3 flex items-center justify-between gap-2">
        <input
          data-testid={`profile-name-${profile.id}`}
          className="min-w-0 flex-1 border-b border-transparent bg-transparent text-sm font-medium text-wc-text-primary outline-none focus:border-wc-accent"
          value={name}
          onChange={(event) => {
            setName(event.target.value);
            setDirty(true);
          }}
        />
        <div className="flex items-center gap-1">
          {dirty && (
            <Button
              data-testid={`profile-save-${profile.id}`}
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={() => {
                onSave({ id: profile.id, scope: profile.scope, name, shortcuts: entries });
                setDirty(false);
              }}
              title={t(strings.settings.shortcutsSection.saveChanges)}
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
            title={t(strings.settings.shortcutsSection.deleteProfile)}
          >
            <Trash2 className="h-3.5 w-3.5 text-wc-text-faint hover:text-wc-error-detail" />
          </Button>
        </div>
      </div>

      <div className="space-y-2">
        {entries.map((entry, index) => (
          <div key={`${profile.id}-${index}`} className="flex items-center gap-2">
            <input
              data-testid={`entry-label-${profile.id}-${index}`}
              placeholder={t(strings.settings.shortcutsSection.labelPlaceholder)}
              className="min-w-0 flex-1 rounded-lg border border-wc-default bg-wc-surface-input px-2 py-1 text-xs text-wc-text-primary outline-none focus:border-wc-accent"
              value={entry.label}
              onChange={(event) => updateEntry(index, "label", event.target.value)}
            />
            <input
              data-testid={`entry-command-${profile.id}-${index}`}
              placeholder={t(strings.settings.shortcutsSection.commandPlaceholder)}
              className="min-w-0 flex-[1.5] rounded-lg border border-wc-default bg-wc-surface-input px-2 py-1 font-mono text-xs text-wc-text-primary outline-none focus:border-wc-accent"
              value={entry.command}
              onChange={(event) => updateEntry(index, "command", event.target.value)}
            />
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 shrink-0"
              onClick={() => removeEntry(index)}
              title={t(strings.settings.shortcutsSection.removeShortcut)}
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
        <Plus className="mr-1 h-3 w-3" />
        {t(strings.settings.shortcutsSection.addShortcut)}
      </Button>
    </div>
  );
}

export default function ShortcutProfilesSection() {
  const { t } = useTranslation();
  const [profiles, setProfiles] = useState<ShortcutProfile[]>([]);
  const [profileError, setProfileError] = useState<string | null>(null);
  const [profileLoading, setProfileLoading] = useState(true);

  const loadProfiles = useCallback(async (signal?: { cancelled: boolean }) => {
    setProfileLoading(true);
    try {
      const resp = await shortcutsClient.listProfiles({});
      if (signal?.cancelled) return;
      setProfiles(resp.profiles);
      setProfileError(null);
    } catch (error) {
      if (signal?.cancelled) return;
      setProfileError(toErrorInfo(error).message);
    } finally {
      if (!signal?.cancelled) {
        setProfileLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    const signal = { cancelled: false };
    void loadProfiles(signal);
    return () => {
      signal.cancelled = true;
    };
  }, [loadProfiles]);

  const handleSaveProfile = useCallback(async (draft: ProfileDraft) => {
    try {
      const resp = await shortcutsClient.upsertProfile(draft);
      if (!resp.profile) {
        throw new Error("upsertProfile: missing profile in response");
      }
      const updated = resp.profile;
      setProfiles((current) => current.map((item) => (item.id === updated.id ? updated : item)));
    } catch (error) {
      setProfileError(toErrorInfo(error).message);
      void loadProfiles();
    }
  }, [loadProfiles]);

  const handleDeleteProfile = useCallback(async (id: string) => {
    try {
      await shortcutsClient.deleteProfile({ id });
      setProfiles((current) => current.filter((item) => item.id !== id));
    } catch (error) {
      setProfileError(toErrorInfo(error).message);
      void loadProfiles();
    }
  }, [loadProfiles]);

  const handleCreateProfile = useCallback(async () => {
    try {
      const resp = await shortcutsClient.upsertProfile({
        id: `profile-${Date.now()}`,
        scope: "workspace",
        name: t(strings.settings.shortcutsSection.newProfileName),
        shortcuts: [{ label: t(strings.settings.shortcutsSection.defaultShortcutLabel), command: "ls -la", description: "" }],
      });
      if (!resp.profile) {
        throw new Error("upsertProfile: missing profile in response");
      }
      const newProfile = resp.profile;
      setProfiles((current) => [...current, newProfile]);
    } catch (error) {
      setProfileError(toErrorInfo(error).message);
    }
  }, [t]);

  return (
    <div className="space-y-4">
      <SettingsSectionIntro
        eyebrow={t(strings.settings.shortcutsSection.eyebrow)}
        title={t(strings.settings.shortcutsSection.title)}
        description={t(strings.settings.shortcutsSection.description)}
      />

      <SettingsCard className="space-y-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-sm font-medium text-wc-text-secondary">{t(strings.settings.shortcutsSection.profilesTitle)}</div>
            <div className="text-[11px] text-wc-text-muted">
              {t(strings.settings.shortcutsSection.profilesHint)}
            </div>
          </div>
          <Button
            data-testid="create-profile"
            variant="outline"
            size="sm"
            className="h-8 px-3 text-xs"
            onClick={handleCreateProfile}
          >
            <Plus className="mr-1 h-3 w-3" />
            {t(strings.settings.shortcutsSection.newProfile)}
          </Button>
        </div>

        {profileError && (
          <div
            data-testid="settings-error"
            className="flex items-start gap-2 rounded-xl border border-wc-error bg-wc-error-surface px-3 py-2 text-xs text-wc-error-detail"
          >
            <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            <span>{profileError}</span>
          </div>
        )}

        {profileLoading ? (
          <div className="py-4 text-center text-xs text-wc-text-faint">{t(strings.settings.shortcutsSection.loading)}</div>
        ) : profiles.length === 0 ? (
          <div className="py-4 text-center text-xs text-wc-text-faint">
            {t(strings.settings.shortcutsSection.empty)}
          </div>
        ) : (
          <div className="space-y-3">
            {profiles.map((profile) => (
              <ShortcutEditor
                key={profile.id}
                profile={profile}
                onSave={handleSaveProfile}
                onDelete={handleDeleteProfile}
              />
            ))}
          </div>
        )}
      </SettingsCard>
    </div>
  );
}
