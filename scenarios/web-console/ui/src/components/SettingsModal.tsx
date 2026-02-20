import { useState, useEffect, useCallback } from "react";
import {
  X,
  GripHorizontal,
  ChevronUp,
  ChevronDown,
  Plus,
  Trash2,
  Save,
  AlertCircle,
  RotateCcw,
  Focus,
} from "lucide-react";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useDraggablePosition } from "../hooks/useDraggablePosition";
import { Button } from "./ui/button";
import ProviderHealthPanel from "./ProviderHealthPanel";
import {
  type ShortcutProfile,
  type ShortcutEntry,
  listShortcutProfiles,
  upsertShortcutProfile,
  deleteShortcutProfile,
  toErrorInfo,
} from "../lib/api";

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

  const updateEntry = (
    idx: number,
    field: keyof ShortcutEntry,
    value: string,
  ) => {
    setEntries((prev) =>
      prev.map((e, i) => (i === idx ? { ...e, [field]: value } : e)),
    );
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
    <div
      data-testid={`shortcut-profile-${profile.id}`}
      className="rounded-lg border border-wc-default bg-wc-surface-input p-3"
    >
      <div className="flex items-center justify-between mb-2">
        <input
          data-testid={`profile-name-${profile.id}`}
          className="bg-transparent text-sm font-medium text-wc-text-primary border-b border-transparent focus:border-wc-accent outline-none"
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setDirty(true);
          }}
        />
        <div className="flex gap-1">
          {dirty && (
            <Button
              data-testid={`profile-save-${profile.id}`}
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={() => {
                onSave({ ...profile, name, shortcuts: entries });
                setDirty(false);
              }}
              title="Save changes"
            >
              <Save className="h-3 w-3 text-green-400" />
            </Button>
          )}
          <Button
            data-testid={`profile-delete-${profile.id}`}
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={() => onDelete(profile.id)}
            title="Delete profile"
          >
            <Trash2 className="h-3 w-3 text-wc-text-faint hover:text-wc-error-detail" />
          </Button>
        </div>
      </div>

      <div className="space-y-1.5">
        {entries.map((entry, idx) => (
          <div key={idx} className="flex items-center gap-1.5">
            <input
              data-testid={`entry-label-${profile.id}-${idx}`}
              placeholder="Label"
              className="flex-1 rounded border border-wc-default bg-wc-surface-base px-2 py-0.5 text-xs text-wc-text-primary outline-none focus:border-wc-accent"
              value={entry.label}
              onChange={(e) => updateEntry(idx, "label", e.target.value)}
            />
            <input
              data-testid={`entry-command-${profile.id}-${idx}`}
              placeholder="Command"
              className="flex-[2] rounded border border-wc-default bg-wc-surface-base px-2 py-0.5 text-xs font-mono text-wc-text-primary outline-none focus:border-wc-accent"
              value={entry.command}
              onChange={(e) => updateEntry(idx, "command", e.target.value)}
            />
            <Button
              variant="ghost"
              size="icon"
              className="h-5 w-5 shrink-0"
              onClick={() => removeEntry(idx)}
            >
              <Trash2 className="h-2.5 w-2.5 text-wc-text-faint" />
            </Button>
          </div>
        ))}
      </div>

      <Button
        variant="ghost"
        size="sm"
        className="mt-1.5 text-xs text-wc-text-faint"
        onClick={addEntry}
      >
        <Plus className="mr-1 h-3 w-3" /> Add shortcut
      </Button>
    </div>
  );
}

export default function SettingsModal() {
  const panes = useWorkspaceStore((s) => s.panes);
  const settingsModalOpen = useWorkspaceStore((s) => s.settingsModalOpen);
  const setSettingsModalOpen = useWorkspaceStore((s) => s.setSettingsModalOpen);
  const movePaneToIndex = useWorkspaceStore((s) => s.movePaneToIndex);
  const removePane = useWorkspaceStore((s) => s.removePane);
  const setActivePane = useWorkspaceStore((s) => s.setActivePane);
  const resetLayout = useWorkspaceStore((s) => s.resetLayout);

  const { elementRef, floatingStyle, pointerHandlers, handleClickCapture } =
    useDraggablePosition({
      isActive: settingsModalOpen,
      storageKey: "wc-settings-pos",
      defaultPosition: () => {
        if (typeof window === "undefined") return { x: 100, y: 100 };
        return {
          x: Math.max(12, (window.innerWidth - 448) / 2),
          y: Math.max(12, window.innerHeight * 0.1),
        };
      },
    });

  // Shortcut profiles state
  const [profiles, setProfiles] = useState<ShortcutProfile[]>([]);
  const [profileError, setProfileError] = useState<string | null>(null);
  const [profileLoading, setProfileLoading] = useState(true);

  const loadProfiles = useCallback(
    async (signal?: { cancelled: boolean }) => {
      setProfileLoading(true);
      try {
        const data = await listShortcutProfiles();
        if (signal?.cancelled) return;
        setProfiles(data);
        setProfileError(null);
      } catch (err) {
        if (signal?.cancelled) return;
        setProfileError(toErrorInfo(err).message);
      } finally {
        if (!signal?.cancelled) setProfileLoading(false);
      }
    },
    [],
  );

  useEffect(() => {
    if (!settingsModalOpen) return;
    const signal = { cancelled: false };
    loadProfiles(signal);
    return () => {
      signal.cancelled = true;
    };
  }, [settingsModalOpen, loadProfiles]);

  const handleSaveProfile = useCallback(
    async (profile: ShortcutProfile) => {
      try {
        const updated = await upsertShortcutProfile({
          id: profile.id,
          scope: profile.scope,
          name: profile.name,
          shortcuts: profile.shortcuts,
        });
        setProfiles((prev) =>
          prev.map((p) => (p.id === updated.id ? updated : p)),
        );
      } catch (err) {
        setProfileError(toErrorInfo(err).message);
        loadProfiles();
      }
    },
    [loadProfiles],
  );

  const handleDeleteProfile = useCallback(
    async (id: string) => {
      try {
        await deleteShortcutProfile(id);
        setProfiles((prev) => prev.filter((p) => p.id !== id));
      } catch (err) {
        setProfileError(toErrorInfo(err).message);
        loadProfiles();
      }
    },
    [loadProfiles],
  );

  const handleCreateProfile = useCallback(async () => {
    try {
      const newProfile = await upsertShortcutProfile({
        id: `profile-${Date.now()}`,
        scope: "workspace",
        name: "New Profile",
        shortcuts: [{ label: "List files", command: "ls -la" }],
      });
      setProfiles((prev) => [...prev, newProfile]);
    } catch (err) {
      setProfileError(toErrorInfo(err).message);
    }
  }, []);

  if (!settingsModalOpen) return null;

  const close = () => setSettingsModalOpen(false);

  return (
    <>
      {/* Backdrop */}
      <div
        data-testid="settings-backdrop"
        className="fixed inset-0 z-40 bg-wc-backdrop"
        onClick={close}
      />

      {/* Modal */}
      <div
        ref={(node) => { elementRef.current = node; }}
        data-testid="settings-modal"
        className="fixed left-0 top-0 z-50 w-[28rem] max-w-[calc(100vw-24px)] max-h-[80vh] overflow-hidden rounded-lg border border-wc-default bg-wc-surface-raised shadow-2xl flex flex-col"
        style={floatingStyle}
        onPointerDown={(e) => {
          const target = e.target as HTMLElement | null;
          const isOnHandle = Boolean(target?.closest("[data-drag-handle]"));
          const isOnControl = Boolean(target?.closest("button, a, input, textarea, select"));
          if (isOnHandle && !isOnControl) {
            pointerHandlers.onPointerDown(e);
          }
        }}
        onPointerMove={pointerHandlers.onPointerMove}
        onPointerUp={pointerHandlers.onPointerUp}
        onPointerCancel={pointerHandlers.onPointerCancel}
        onClickCapture={handleClickCapture}
      >
        {/* Drag handle header */}
        <div
          data-drag-handle
          className="flex items-center justify-between px-4 py-2 border-b border-wc-default cursor-grab active:cursor-grabbing select-none"
        >
          <div className="flex items-center gap-2">
            <GripHorizontal className="h-4 w-4 text-wc-text-faint" />
            <h2 className="text-sm font-semibold text-wc-text-primary">
              Settings
            </h2>
          </div>
          <Button
            data-testid="settings-close"
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={close}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto p-4 space-y-5">
          {/* Section 1: Workspace Management */}
          <section>
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted">
                Workspace
              </h3>
              <Button
                data-testid="reset-layout"
                variant="ghost"
                size="sm"
                className="text-xs text-wc-text-faint"
                onClick={resetLayout}
                title="Reset grid layout"
              >
                <RotateCcw className="mr-1 h-3 w-3" /> Reset Layout
              </Button>
            </div>

            {panes.length === 0 ? (
              <div className="text-center text-xs text-wc-text-faint py-2">
                No terminals open
              </div>
            ) : (
              <div className="space-y-1">
                {panes.map((pane, idx) => (
                  <div
                    key={pane.sessionId}
                    data-testid={`settings-pane-${pane.sessionId}`}
                    className="flex items-center gap-1.5 rounded border border-wc-default bg-wc-surface-input px-2 py-1"
                  >
                    {/* Color indicator */}
                    <div
                      className="h-3 w-3 rounded-full shrink-0 border border-wc-default"
                      style={{
                        backgroundColor:
                          pane.headerColor !== "transparent"
                            ? pane.headerColor
                            : "rgb(var(--wc-surface-input))",
                      }}
                    />
                    <span className="flex-1 text-xs text-wc-text-secondary truncate">
                      {pane.name}
                    </span>

                    {/* Reorder buttons */}
                    <Button
                      data-testid={`settings-pane-up-${pane.sessionId}`}
                      variant="ghost"
                      size="icon"
                      className="h-5 w-5"
                      disabled={idx === 0}
                      onClick={() =>
                        movePaneToIndex(pane.sessionId, idx - 1)
                      }
                      title="Move up"
                    >
                      <ChevronUp className="h-3 w-3" />
                    </Button>
                    <Button
                      data-testid={`settings-pane-down-${pane.sessionId}`}
                      variant="ghost"
                      size="icon"
                      className="h-5 w-5"
                      disabled={idx === panes.length - 1}
                      onClick={() =>
                        movePaneToIndex(pane.sessionId, idx + 1)
                      }
                      title="Move down"
                    >
                      <ChevronDown className="h-3 w-3" />
                    </Button>
                    <Button
                      data-testid={`settings-pane-focus-${pane.sessionId}`}
                      variant="ghost"
                      size="icon"
                      className="h-5 w-5"
                      onClick={() => {
                        setActivePane(pane.sessionId);
                        close();
                      }}
                      title="Focus terminal"
                    >
                      <Focus className="h-3 w-3" />
                    </Button>
                    <Button
                      data-testid={`settings-pane-remove-${pane.sessionId}`}
                      variant="ghost"
                      size="icon"
                      className="h-5 w-5"
                      onClick={() => removePane(pane.sessionId)}
                      title="Close terminal"
                    >
                      <X className="h-3 w-3 text-wc-text-faint" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </section>

          {/* Section 2: Shortcut Profiles */}
          <section>
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted">
                Shortcut Profiles
              </h3>
              <Button
                data-testid="create-profile"
                variant="outline"
                size="sm"
                className="text-xs"
                onClick={handleCreateProfile}
              >
                <Plus className="mr-1 h-3 w-3" /> New Profile
              </Button>
            </div>

            {profileError && (
              <div
                data-testid="settings-error"
                className="flex items-center gap-2 rounded border border-wc-error bg-wc-error-surface px-2 py-1.5 text-xs text-wc-error-detail mb-2"
              >
                <AlertCircle className="h-3 w-3 shrink-0" />
                {profileError}
              </div>
            )}

            {profileLoading ? (
              <div className="text-center text-xs text-wc-text-faint py-2">
                Loading...
              </div>
            ) : profiles.length === 0 ? (
              <div className="text-center text-xs text-wc-text-faint py-2">
                No shortcut profiles configured
              </div>
            ) : (
              <div className="space-y-2">
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
          </section>

          {/* Section 3: AI Providers */}
          <section>
            <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
              AI Providers
            </h3>
            <div className="rounded-lg border border-wc-default bg-wc-surface-input p-3">
              <ProviderHealthPanel open={settingsModalOpen} />
            </div>
          </section>
        </div>
      </div>
    </>
  );
}
