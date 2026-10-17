import { useEffect, useState } from "react";
import { Pin, PinOff, Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { MasterDetail } from "@vrooli/react-component-library/MasterDetail/1";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

import type { SnippetDTO } from "../../api/snippets";
import { HEADER_COLORS } from "../../consts/config";
import { strings } from "../../consts/strings";
import { useSnippets } from "../../hooks/useSnippets";
import { cn } from "../../lib/classnames";
import { SnippetBodyEditor } from "../snippets/SnippetBodyEditor";
import { SnippetSaveSheet } from "../snippets/SnippetSaveSheet";
import { Button } from "../ui/button";

import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";

// [REQ:P0-015g] Personal snippet management

export default function SnippetsPanel() {
  const { t } = useTranslation();
  const { snippets, status, save, remove, promote } = useSnippets();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [draft, setDraft] = useState<SnippetDTO | null>(null);
  const [creating, setCreating] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [promoteOpen, setPromoteOpen] = useState(false);
  const [promoting, setPromoting] = useState(false);
  const [promotionResult, setPromotionResult] = useState("");
  const [promotionError, setPromotionError] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<SnippetDTO | null>(null);

  useEffect(() => {
    setDraft(snippets.find((snippet) => snippet.id === selectedId) ?? null);
  }, [selectedId, snippets]);

  const saveDraft = async () => {
    if (!draft || saving || !draft.name.trim()) return;
    setSaving(true);
    setError("");
    try {
      const saved = await save({
        id: draft.id,
        name: draft.name,
        body: draft.body,
        color: draft.color,
        pinned: draft.pinned,
        sort_order: draft.sort_order,
      });
      setSelectedId(saved.id);
    } catch {
      setError(t(strings.snippets.settings.saveError));
    } finally {
      setSaving(false);
    }
  };

  const togglePin = async (snippet: SnippetDTO) => {
    setError("");
    try {
      await save({
        id: snippet.id,
        name: snippet.name,
        body: snippet.body,
        color: snippet.color,
        pinned: !snippet.pinned,
        sort_order: snippet.sort_order,
      });
    } catch {
      setError(t(strings.snippets.settings.pinError));
    }
  };

  // A browser `confirm()` blocks the whole page, cannot be styled, and on a
  // touch device is dismissed by gestures this app also uses. Deletion is the
  // one irreversible action here, so it gets the same governed confirmation
  // every other destructive path in the console uses.
  const confirmDelete = async () => {
    const snippet = deleteTarget;
    if (!snippet) return;
    setError("");
    try {
      if (await remove(snippet.id)) {
        setSelectedId((current) => (current === snippet.id ? null : current));
      }
      setDeleteTarget(null);
    } catch {
      setError(t(strings.snippets.settings.deleteError));
      setDeleteTarget(null);
    }
  };

  const items = snippets.map((snippet) => ({
    id: snippet.id,
    title: snippet.name,
    summary: snippet.body.replace(/\s+/g, " "),
    meta: t(strings.snippets.settings.usedCount, { count: snippet.use_count }),
    value: snippet,
  }));

  const promoteDraft = async () => {
    if (!draft || promoting) return;
    setPromoting(true);
    setPromotionError("");
    try {
      const identifier = await promote(draft.id);
      setPromotionResult(identifier);
    } catch (promotionFailure) {
      const message = promotionFailure instanceof Error ? promotionFailure.message : "";
      setPromotionError(message === "prompt-manager is not available on this host"
        ? t(strings.snippets.settings.promptManagerUnavailable)
        : message || t(strings.snippets.settings.promoteError));
    } finally {
      setPromoting(false);
    }
  };

  return (
    <SettingsList data-testid="snippets-panel">
      <SettingsList.Intro
        eyebrow={t(strings.snippets.settings.eyebrow)}
        title={t(strings.snippets.settings.title)}
        description={t(strings.snippets.settings.description)}
      />

      <div className="flex justify-end">
        <Button data-testid="snippets-create" variant="outline" size="sm" onClick={() => { setCreating(true); }}>
          <Plus className="me-1.5 h-4 w-4" aria-hidden />
          {t(strings.snippets.settings.new)}
        </Button>
      </div>

      {snippets.length === 0 && status === "ready" && (
        <p data-testid="snippets-empty" className="py-6 text-center text-sm text-wc-text-muted">
          {t(strings.snippets.settings.empty)}
        </p>
      )}

      <MasterDetail
        items={items}
        selectedId={selectedId}
        onSelect={(item) => { setSelectedId(item.id); }}
        status={status === "loading" ? "loading" : status === "request-error" ? "request-error" : "default"}
        label={t(strings.snippets.settings.title)}
        renderMaster={(item, state) => {
          const snippet = item.value;
          return (
            <div data-testid={`snippet-settings-row-${item.id}`} className="flex items-center gap-2">
              <span className="flex min-w-0 flex-1 items-center gap-2 text-start" onClick={state.select}>
                <span className="h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: snippet.color }} aria-hidden />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm text-wc-text-primary">{item.title}</span>
                  <span className="block truncate text-[11px] text-wc-text-faint">{item.meta}</span>
                </span>
              </span>
              <span
                role="button"
                tabIndex={0}
                data-testid={`snippet-settings-pin-${item.id}`}
                aria-label={t(snippet.pinned ? strings.snippets.settings.unpin : strings.snippets.settings.pin)}
                aria-pressed={snippet.pinned}
                className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-wc-text-muted hover:bg-wc-surface-input"
                onClick={(event) => { event.stopPropagation(); void togglePin(snippet); }}
                onKeyDown={(event) => {
                  if (event.key !== "Enter" && event.key !== " ") return;
                  event.preventDefault();
                  event.stopPropagation();
                  void togglePin(snippet);
                }}
              >
                {snippet.pinned ? <PinOff className="h-4 w-4" /> : <Pin className="h-4 w-4" />}
              </span>
              <span
                role="button"
                tabIndex={0}
                data-testid={`snippet-settings-delete-${item.id}`}
                aria-label={t(strings.snippets.settings.delete)}
                className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-red-400 hover:bg-red-500/10"
                onClick={(event) => { event.stopPropagation(); setDeleteTarget(snippet); }}
                onKeyDown={(event) => {
                  if (event.key !== "Enter" && event.key !== " ") return;
                  event.preventDefault();
                  event.stopPropagation();
                  setDeleteTarget(snippet);
                }}
              >
                <Trash2 className="h-4 w-4" />
              </span>
            </div>
          );
        }}
        renderDetail={() => null}
      />

      {draft && (
        <div data-testid="snippet-settings-detail">
        <SettingsList.Group>
          <label className="block space-y-1.5 text-xs font-medium text-wc-text-secondary">
            <span>{t(strings.snippets.name)}</span>
            <input
              data-testid="snippet-settings-name"
              value={draft.name}
              onChange={(event) => { setDraft({ ...draft, name: event.target.value }); }}
              className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary outline-none focus:border-wc-accent"
            />
          </label>

          <div>
            <span className="text-xs font-medium text-wc-text-secondary">{t(strings.snippets.color)}</span>
            <div className="flex flex-wrap gap-1.5">
              {HEADER_COLORS.map((color) => (
                <button
                  key={color}
                  type="button"
                  data-testid={`snippet-settings-color-${color}`}
                  aria-label={t(strings.snippets.chooseColor, { color })}
                  aria-pressed={draft.color === color}
                  className={cn(
                    "h-11 w-11 rounded-full border border-wc-default transition hover:scale-105",
                    draft.color === color && "ring-2 ring-wc-accent",
                  )}
                  style={{ backgroundColor: color }}
                  onClick={() => { setDraft({ ...draft, color }); }}
                />
              ))}
            </div>
          </div>

          <label className="block space-y-1.5 text-xs font-medium text-wc-text-secondary">
            <span>{t(strings.snippets.body)}</span>
            <SnippetBodyEditor
              body={draft.body}
              onChange={(body) => { setDraft({ ...draft, body }); }}
              bodyTestId="snippet-settings-body"
              countTestId="snippet-settings-variable-count"
            />
          </label>

          {error && <p data-testid="snippet-settings-error" className="text-xs text-red-400">{error}</p>}
          <div className="flex justify-end gap-2">
            <Button
              data-testid="snippet-settings-promote"
              variant="outline"
              size="sm"
              onClick={() => {
                setPromotionResult("");
                setPromotionError("");
                setPromoteOpen(true);
              }}
            >
              {t(strings.snippets.settings.promoteTitle)}
            </Button>
            <Button data-testid="snippet-settings-save" size="sm" disabled={saving || !draft.name.trim()} onClick={() => void saveDraft()}>
              {t(saving ? strings.snippets.save.saving : strings.snippets.save.save)}
            </Button>
          </div>
        </SettingsList.Group>
        </div>
      )}

      <SnippetSaveSheet
        open={creating}
        onClose={() => { setCreating(false); }}
        mode="create"
        initialBody=""
        sourceLabel={t(strings.snippets.save.createdInSettings)}
        onSaved={(snippet) => { setSelectedId(snippet.id); }}
      />

      <ResponsiveDialog
        open={promoteOpen}
        onClose={() => { setPromoteOpen(false); }}
        size="sm"
        title={t(strings.snippets.settings.promoteTitle)}
        closeLabel={t(strings.snippets.close)}
        testId="snippet-promote-dialog"
        avoidKeyboard
      >
        <div className="p-4">
          <p>{t(strings.snippets.settings.promoteFactGoverned)}</p>
          <p>{t(strings.snippets.settings.promoteFactSnippetStays)}</p>
          <p>{t(strings.snippets.settings.promoteFactNoSync)}</p>
          {promotionResult && (
            <p data-testid="snippet-promote-success" className="text-sm text-emerald-400">
              {t(strings.snippets.settings.promoteSuccess, { identifier: promotionResult })}
            </p>
          )}
          {promotionError && (
            <p data-testid="snippet-promote-error" className="text-sm text-red-400">
              {promotionError}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <Button data-testid="snippet-promote-cancel" variant="outline" size="sm" onClick={() => { setPromoteOpen(false); }}>
              {t(promotionResult ? strings.snippets.close : strings.snippets.cancel)}
            </Button>
            {!promotionResult && (
              <Button data-testid="snippet-promote-confirm" size="sm" disabled={promoting} onClick={() => void promoteDraft()}>
                {t(promoting ? strings.snippets.settings.promotePending : strings.snippets.settings.promote)}
              </Button>
            )}
          </div>
        </div>
      </ResponsiveDialog>

      {deleteTarget && (
        <AlertDialog
          open
          destructive
          title={t(strings.snippets.settings.delete)}
          description={t(strings.snippets.settings.deleteConfirm, { name: deleteTarget.name })}
          cancelLabel={t(strings.snippets.cancel)}
          confirmLabel={t(strings.snippets.settings.delete)}
          onCancel={() => { setDeleteTarget(null); }}
          onConfirm={() => void confirmDelete()}
          testIdPrefix="snippet-delete"
        />
      )}
    </SettingsList>
  );
}
