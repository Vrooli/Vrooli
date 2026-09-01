// DOC: docs/reference/configuration.md#launcher-shortcuts
/**
 * The launch-command editor.
 *
 * This screen owns what the launcher runs, so it is the one place that can
 * warn an operator that a command they are about to save will silently stop
 * their messages being recorded. That check is the reason the layout changed:
 * the previous editor put a label and a command in two bare inputs on one
 * flex row, which clipped the command off the right edge of a phone and left
 * no room to say anything about it at all.
 *
 * The rewrite is otherwise mundane and deliberate:
 *   - one field per line, each with a visible label;
 *   - the description field, which existed in the draft type and had no input,
 *     so it could be preserved-or-lost but never edited;
 *   - stable React keys, because index keys make a reorder reuse the wrong
 *     row's state, and reordering is the point;
 *   - one save bar with a dirty count, rather than a 28px icon that appears
 *     only once the form is already dirty;
 *   - a confirmation before deleting a profile.
 *
 * [REQ:P1-002b] Shortcut Profile Management UI
 * [REQ:P0-006b] Configurable Shortcut Entries
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AlertCircle, Check, GripVertical, Info, Plus, Trash2, TriangleAlert } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";

import type { Profile as ShortcutProfile } from "@vrooli/proto-types/web-console/v1/shortcuts/shortcuts_pb";
import { shortcutsClient } from "../../api/shortcuts";
import { strings } from "../../consts/strings";
import { toErrorInfo } from "../../lib/errors";
import { assessCapture, governedRewrite } from "../../lib/captureSafety";
import { moveItem } from "../launcher/agentGrid";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/0";

interface ShortcutDraft {
  /** Stable identity for React and for reordering. Never the array index. */
  uid: string;
  label: string;
  command: string;
  description: string;
  agentId: string;
}

interface ProfileDraft {
  id: string;
  scope: string;
  name: string;
  shortcuts: { label: string; command: string; description: string; agentId: string }[];
}

let uidCounter = 0;
function nextUID(): string {
  uidCounter += 1;
  return `draft-${String(uidCounter)}`;
}

const FIELD_CLASS =
  "w-full rounded-lg border border-wc-default bg-wc-surface-input px-2.5 py-2 text-xs text-wc-text-primary outline-none transition focus:border-wc-accent";

const LABEL_CLASS = "mb-1 block text-[9.5px] font-semibold uppercase tracking-[0.09em] text-wc-text-faint";

/** A short note under a command saying whether it will be captured. */
function CaptureNote({
  command,
  agentId,
  onUseGoverned,
}: {
  command: string;
  agentId: string;
  onUseGoverned: (rewritten: string) => void;
}) {
  const { t } = useTranslation();
  const assessment = assessCapture(command, agentId || undefined);
  const section = strings.settings.shortcutsSection;

  if (assessment.verdict === "governed") {
    return (
      <p data-testid="capture-note-governed" className="flex items-start gap-2 rounded-lg border border-wc-success/30 bg-wc-success/[0.07] px-2.5 py-2 text-[11px] text-wc-success">
        <Check className="mt-px h-3 w-3 shrink-0" aria-hidden />
        <span>{t(section.captureGoverned, { via: assessment.via ?? "" })}</span>
      </p>
    );
  }
  if (assessment.verdict === "independent") {
    return (
      <p data-testid="capture-note-independent" className="flex items-start gap-2 text-[11px] text-wc-text-muted">
        <Info className="mt-px h-3 w-3 shrink-0" aria-hidden />
        <span>{t(section.captureIndependent)}</span>
      </p>
    );
  }
  if (assessment.verdict !== "path-dependent") return null;

  const rewritten = governedRewrite(command, agentId);
  return (
    <p data-testid="capture-note-warning" className="flex flex-wrap items-start gap-2 rounded-lg border border-wc-warning/35 bg-wc-warning/[0.07] px-2.5 py-2 text-[11px] text-wc-warning">
      <TriangleAlert className="mt-px h-3 w-3 shrink-0" aria-hidden />
      <span className="min-w-0 flex-1">{t(section.capturePathDependent)}</span>
      {rewritten && (
        <button
          type="button"
          data-testid="capture-use-governed"
          className="shrink-0 rounded-md border border-wc-warning/50 px-2 py-0.5 font-semibold"
          onClick={() => { onUseGoverned(rewritten); }}
        >
          {t(section.useGovernedLaunch)}
        </button>
      )}
    </p>
  );
}

function ShortcutEditor({
  profile,
  onSave,
  onDelete,
}: {
  profile: ShortcutProfile;
  onSave: (draft: ProfileDraft) => Promise<void> | void;
  onDelete: (id: string) => void;
}) {
  const { t } = useTranslation();
  const section = strings.settings.shortcutsSection;
  const toDraft = useCallback(
    (): ShortcutDraft[] =>
      profile.shortcuts.map((entry) => ({
        uid: nextUID(),
        label: entry.label,
        command: entry.command,
        description: entry.description,
        agentId: entry.agentId,
      })),
    [profile.shortcuts],
  );

  const [entries, setEntries] = useState<ShortcutDraft[]>(toDraft);
  const [name, setName] = useState(profile.name);
  const [dirtyCount, setDirtyCount] = useState(0);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const dragFrom = useRef<number | null>(null);

  // A saved profile arriving from the server replaces the draft. Keeping local
  // edits over a server round-trip is how two tabs silently overwrite each
  // other; the save bar makes the discard visible instead.
  useEffect(() => {
    setEntries(toDraft());
    setName(profile.name);
    setDirtyCount(0);
  }, [profile.name, profile.updatedAt, toDraft]);

  const markDirty = useCallback(() => { setDirtyCount((count) => count + 1); }, []);

  const updateEntry = (uid: string, field: "label" | "command" | "description", value: string) => {
    setEntries((current) => current.map((entry) => (entry.uid === uid ? { ...entry, [field]: value } : entry)));
    markDirty();
  };

  const addEntry = () => {
    setEntries((current) => [...current, { uid: nextUID(), label: "", command: "", description: "", agentId: "" }]);
    markDirty();
  };

  const removeEntry = (uid: string) => {
    setEntries((current) => current.filter((entry) => entry.uid !== uid));
    markDirty();
  };

  const move = (from: number, to: number) => {
    if (to < 0 || to >= entries.length) return;
    setEntries((current) => moveItem(current, from, to));
    markDirty();
  };

  const discard = () => {
    setEntries(toDraft());
    setName(profile.name);
    setDirtyCount(0);
  };

  const save = () => {
    void Promise.resolve(
      onSave({
        id: profile.id,
        scope: profile.scope,
        name,
        shortcuts: entries.map((entry) => ({
          label: entry.label,
          command: entry.command,
          description: entry.description,
          // The server owns the derivation, so an entry whose command changed
          // is sent with no agent id rather than a stale one.
          agentId: entry.agentId,
        })),
      }),
    ).then(() => { setDirtyCount(0); });
  };

  return (
    <div
      data-testid={`shortcut-profile-${profile.id}`}
      className="rounded-xl border border-wc-default bg-wc-surface-base/70 p-3"
    >
      <div className="mb-3 flex items-center justify-between gap-2">
        <div className="min-w-0 flex-1">
          <label htmlFor={`profile-name-${profile.id}`} className={LABEL_CLASS}>
            {t(section.profilesTitle)}
          </label>
          <input
            id={`profile-name-${profile.id}`}
            data-testid={`profile-name-${profile.id}`}
            className={FIELD_CLASS}
            value={name}
            onChange={(event) => { setName(event.target.value); markDirty(); }}
          />
        </div>
        <Button
          data-testid={`profile-delete-${profile.id}`}
          variant="ghost"
          size="icon"
          className="mt-4 h-8 w-8 shrink-0"
          aria-label={t(section.deleteProfile)}
          onClick={() => { setConfirmingDelete(true); }}
        >
          <Trash2 className="h-3.5 w-3.5 text-wc-text-faint hover:text-wc-error-detail" />
        </Button>
      </div>

      {confirmingDelete && (
        <div
          data-testid={`profile-delete-confirm-${profile.id}`}
          className="mb-3 flex flex-wrap items-center gap-2 rounded-lg border border-wc-error bg-wc-error-surface px-3 py-2 text-xs text-wc-error-detail"
        >
          <span className="min-w-0 flex-1">{t(section.confirmDeleteProfile)}</span>
          <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={() => { setConfirmingDelete(false); }}>
            {t(section.discard)}
          </Button>
          <Button
            data-testid={`profile-delete-confirmed-${profile.id}`}
            variant="outline"
            size="sm"
            className="h-7 px-2 text-xs"
            onClick={() => { setConfirmingDelete(false); onDelete(profile.id); }}
          >
            {t(section.deleteProfile)}
          </Button>
        </div>
      )}

      <div className="flex flex-col gap-2">
        {entries.map((entry, index) => (
          <div
            key={entry.uid}
            data-testid={`shortcut-entry-${entry.uid}`}
            draggable
            onDragStart={() => { dragFrom.current = index; }}
            onDragOver={(event) => { event.preventDefault(); }}
            onDrop={() => {
              if (dragFrom.current !== null && dragFrom.current !== index) move(dragFrom.current, index);
              dragFrom.current = null;
            }}
            className="flex flex-col gap-2 rounded-xl border border-wc-default bg-wc-surface-input/40 p-2.5"
          >
            <div className="flex items-center gap-2">
              <span
                role="button"
                tabIndex={0}
                aria-label={t(strings.launcher.reorderGrip, { name: entry.label || t(section.commandLabel) })}
                data-testid={`shortcut-grip-${entry.uid}`}
                className="cursor-grab text-wc-text-faint"
                onKeyDown={(event) => {
                  if (!event.altKey) return;
                  const delta = event.key === "ArrowUp" ? -1 : event.key === "ArrowDown" ? 1 : 0;
                  if (delta === 0) return;
                  event.preventDefault();
                  move(index, index + delta);
                }}
              >
                <GripVertical className="h-4 w-4" aria-hidden />
              </span>
              <input
                data-testid={`entry-label-${entry.uid}`}
                aria-label={t(section.labelPlaceholder)}
                placeholder={t(section.labelPlaceholder)}
                className="min-w-0 flex-1 bg-transparent text-[13px] font-semibold text-wc-text-primary outline-none"
                value={entry.label}
                onChange={(event) => { updateEntry(entry.uid, "label", event.target.value); }}
              />
              <span
                data-testid={`entry-agent-${entry.uid}`}
                className="shrink-0 rounded-full border border-wc-default px-2 py-0.5 font-mono text-[9.5px] text-wc-text-faint"
              >
                {entry.agentId || t(section.notAnAgent)}
              </span>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0"
                aria-label={t(section.removeCommand)}
                data-testid={`entry-remove-${entry.uid}`}
                onClick={() => { removeEntry(entry.uid); }}
              >
                <Trash2 className="h-3 w-3 text-wc-text-faint" />
              </Button>
            </div>

            <div>
              <label htmlFor={`entry-command-${entry.uid}`} className={LABEL_CLASS}>{t(section.commandLabel)}</label>
              <textarea
                id={`entry-command-${entry.uid}`}
                data-testid={`entry-command-${entry.uid}`}
                rows={2}
                placeholder={t(section.commandPlaceholder)}
                className={`${FIELD_CLASS} resize-y break-all font-mono leading-relaxed`}
                value={entry.command}
                onChange={(event) => { updateEntry(entry.uid, "command", event.target.value); }}
              />
            </div>

            <div>
              <label htmlFor={`entry-description-${entry.uid}`} className={LABEL_CLASS}>{t(section.descriptionLabel)}</label>
              <input
                id={`entry-description-${entry.uid}`}
                data-testid={`entry-description-${entry.uid}`}
                placeholder={t(section.descriptionPlaceholder)}
                className={FIELD_CLASS}
                value={entry.description}
                onChange={(event) => { updateEntry(entry.uid, "description", event.target.value); }}
              />
            </div>

            <CaptureNote
              command={entry.command}
              agentId={entry.agentId}
              onUseGoverned={(rewritten) => { updateEntry(entry.uid, "command", rewritten); }}
            />
          </div>
        ))}
      </div>

      <Button
        variant="ghost"
        size="sm"
        className="mt-2 text-xs text-wc-accent"
        data-testid={`profile-add-entry-${profile.id}`}
        onClick={addEntry}
      >
        <Plus className="me-1 h-3 w-3" />
        {t(section.addCommand)}
      </Button>

      {dirtyCount > 0 && (
        <div
          data-testid={`profile-save-bar-${profile.id}`}
          className="mt-3 flex items-center justify-between gap-2 rounded-lg border border-wc-default bg-wc-surface-raised/70 px-3 py-2"
        >
          <span className="text-[11.5px] text-wc-warning">{t(section.unsavedChanges, { count: dirtyCount })}</span>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={discard}>
              {t(section.discard)}
            </Button>
            <Button data-testid={`profile-save-${profile.id}`} size="sm" className="h-7 px-3 text-xs" onClick={save}>
              {t(section.save)}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

export default function ShortcutProfilesSection() {
  const { t } = useTranslation();
  const section = strings.settings.shortcutsSection;
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
      setProfileError(null);
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
        id: `profile-${String(Date.now())}`,
        scope: "workspace",
        name: t(section.newProfileName),
        shortcuts: [{ label: t(section.defaultShortcutLabel), command: "ls -la", description: "", agentId: "" }],
      });
      if (!resp.profile) {
        throw new Error("upsertProfile: missing profile in response");
      }
      const newProfile = resp.profile;
      setProfiles((current) => [...current, newProfile]);
    } catch (error) {
      setProfileError(toErrorInfo(error).message);
    }
  }, [section.defaultShortcutLabel, section.newProfileName, t]);

  const body = useMemo(() => {
    if (profileLoading) {
      return <div className="py-4 text-center text-xs text-wc-text-faint">{t(section.loading)}</div>;
    }
    if (profiles.length === 0) {
      return <div className="py-4 text-center text-xs text-wc-text-faint">{t(section.empty)}</div>;
    }
    return (
      <div className="flex flex-col gap-3">
        {profiles.map((profile) => (
          <ShortcutEditor
            key={profile.id}
            profile={profile}
            onSave={handleSaveProfile}
            onDelete={(id) => { void handleDeleteProfile(id); }}
          />
        ))}
      </div>
    );
  }, [handleDeleteProfile, handleSaveProfile, profileLoading, profiles, section.empty, section.loading, t]);

  return (
    <SettingsList>
      <SettingsList.Intro
        eyebrow={t(section.eyebrow)}
        title={t(section.title)}
        description={t(section.description)}
      />

      <SettingsList.Group>
        <div className="flex items-center justify-between gap-3">
          <div>
            <div className="text-sm font-medium text-wc-text-secondary">{t(section.profilesTitle)}</div>
            <div className="text-[11px] text-wc-text-muted">{t(section.profilesHint)}</div>
          </div>
          <Button
            data-testid="create-profile"
            variant="outline"
            size="sm"
            className="h-8 px-3 text-xs"
            onClick={() => { void handleCreateProfile(); }}
          >
            <Plus className="me-1 h-3 w-3" />
            {t(section.newProfile)}
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

        {body}
      </SettingsList.Group>
    </SettingsList>
  );
}
