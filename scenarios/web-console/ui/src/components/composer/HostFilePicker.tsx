import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertTriangle, ArrowLeft, Check, FileText, Folder, Loader2, RotateCw, ScanEye } from "lucide-react";
import { useTranslation } from "react-i18next";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";

import MessagesFileViewer, { PreviewBreadcrumbs } from "../MessagesFileViewer";
import { DirectoryPreview } from "../file-preview/renderers/DirectoryPreview";
import { useFilePreviewController } from "../file-preview/useFilePreviewController";
import type { DirectorySelection } from "../file-preview/types";
import { canThumbnail, previewThumbnailHref, resolveFilePreview, type PreviewModel } from "../../api/filePreview";
import type { HostAttachmentPick } from "../../hooks/useComposerAttachments";
import { mentionedFilePaths } from "../../lib/fileReferences";
import { basename, pathCrumbs } from "../../lib/paths";
import { cn } from "../../lib/classnames";
import { strings } from "../../consts/strings";
import { useConversationStore } from "../../stores/useConversationStore";

type PickerTab = "session" | "home" | "mentioned";
const TAB_ORDER: readonly PickerTab[] = ["session", "home", "mentioned"];

interface HostFilePickerProps {
  open: boolean;
  /** Session whose machine is browsed; relative starts resolve against its cwd. */
  sessionId: string;
  onClose: () => void;
  onAttach: (picks: HostAttachmentPick[]) => void;
}

// startPathFor resolves a tab's starting folder to an absolute path. "." can
// match both the session cwd and the project root; the session cwd is the one
// the operator means.
async function startPathFor(sessionId: string, tab: "session" | "home"): Promise<string> {
  const model = await resolveFilePreview(sessionId, tab === "home" ? "~" : ".");
  const fromCwd = model.candidates?.find((c) => c.resolutionBasis === "session_cwd") ?? model.candidates?.[0];
  return model.resolvedPath || fromCwd?.resolvedPath || (tab === "home" ? "~" : ".");
}

/**
 * HostFilePicker — attach files and folders that already live on the machine
 * the session runs on. It is the file-preview folder browser in pick mode: the
 * same controller walks folders, rows gain checkboxes, and the picks are sent
 * by absolute path, so nothing is copied or uploaded.
 */
export default function HostFilePicker({ open, sessionId, onClose, onAttach }: HostFilePickerProps) {
  const { t } = useTranslation();
  const controller = useFilePreviewController(sessionId);
  const { state, openPreview, navigateTo, navigateBack, loadMore, setListOptions, reportError, close: closeController } = controller;
  const [tab, setTab] = useState<PickerTab>("session");
  const [selected, setSelected] = useState<Map<string, HostAttachmentPick>>(() => new Map());
  const [startError, setStartError] = useState<string | null>(null);
  // A file opened to check it before picking. It opens in the regular viewer
  // over the picker; closing it returns here with the selection intact.
  const [previewPath, setPreviewPath] = useState<string | null>(null);

  const openTab = useCallback(async (next: "session" | "home") => {
    setStartError(null);
    try {
      void openPreview(await startPathFor(sessionId, next));
    } catch (err) {
      setStartError(err instanceof Error && err.message ? err.message : t(strings.composer.hostPickerUnavailable));
    }
  }, [openPreview, sessionId, t]);

  // Every open starts fresh in the session folder with nothing selected.
  useEffect(() => {
    if (!open) { closeController(); return; }
    setTab("session");
    setSelected(new Map());
    setPreviewPath(null);
    void openTab("session");
  }, [open, openTab, closeController]);

  const changeTab = useCallback((next: string) => {
    const value = next as PickerTab;
    setTab(value);
    if (value !== "mentioned") void openTab(value);
  }, [openTab]);

  const toggle = useCallback((pick: HostAttachmentPick) => {
    setSelected((prev) => {
      const next = new Map(prev);
      if (next.has(pick.path)) next.delete(pick.path);
      else next.set(pick.path, pick);
      return next;
    });
  }, []);

  const selection = useMemo<DirectorySelection>(() => ({
    isSelected: (path) => selected.has(path),
    onToggle: (path, entry) => { toggle({ path, isDirectory: entry.entryType === "directory" || entry.kind === "directory", sizeBytes: entry.sizeBytes }); },
    toggleLabel: (name) => t(strings.composer.hostPickerToggle, { name }),
    onPreview: setPreviewPath,
    previewLabel: (name) => t(strings.composer.hostPickerPreview, { name }),
  }), [selected, t, toggle]);

  const attach = useCallback(() => {
    if (selected.size === 0) return;
    onAttach(Array.from(selected.values()));
    onClose();
  }, [onAttach, onClose, selected]);

  const { status, model, listing, stack, error, loadingMore } = state;
  const listingRefreshing = status === "loadingListing" && Boolean(listing);
  const loading = !startError && (status === "idle" || status === "resolving" || (status === "loadingListing" && !listingRefreshing));
  const crumbs = model?.kind === "directory" && model.resolvedPath ? pathCrumbs(model.resolvedPath) : [];
  const browsing = tab !== "mentioned";

  const subheader = (
    <div className="flex min-w-0 flex-col border-b border-wc-default">
      <Tabs
        ariaLabel={t(strings.composer.hostPickerTitle)}
        active={tab}
        onChange={changeTab}
        variant="segmented"
        density="compact"
        testId="host-picker-tabs"
        itemTestId={(item) => `host-picker-tab-${item}`}
        items={TAB_ORDER.map((id) => ({ id, label: t(TAB_LABEL[id]) }))}
      />
      {browsing && (
        <nav aria-label={t(strings.documentPreview.navigation)} className="flex min-w-0 items-center gap-1 bg-wc-surface-raised px-2 py-1">
          <IconButton type="button" onClick={navigateBack} disabled={stack.length === 0} data-testid="host-picker-back" aria-label={t(strings.documentPreview.back)} title={t(strings.documentPreview.back)} surface="ghost" shape="rounded" className="shrink-0"><ArrowLeft className="rtl:rotate-180" /></IconButton>
          {crumbs.length > 0 && <PreviewBreadcrumbs crumbs={crumbs} onNavigate={navigateTo} />}
        </nav>
      )}
    </div>
  );

  // The drawer's footer is already a padded, bordered row; these are its items.
  const footer = (
    <>
      <p className="min-w-0 flex-1 self-center text-xs text-wc-text-muted" data-testid="host-picker-summary">
        {selected.size > 0 ? t(strings.composer.hostPickerSelected, { count: selected.size }) : t(strings.composer.hostPickerHint)}
      </p>
      <button
        type="button"
        onClick={attach}
        disabled={selected.size === 0}
        data-testid="host-picker-attach"
        className="inline-flex min-h-11 shrink-0 items-center gap-1.5 rounded-lg bg-wc-accent px-4 text-sm font-medium text-white transition disabled:opacity-50"
      >
        {selected.size > 0 ? t(strings.composer.hostPickerAttachCount, { count: selected.size }) : t(strings.composer.hostPickerAttach)}
      </button>
    </>
  );

  let content: React.JSX.Element;
  if (!browsing) {
    content = <MentionedList open={open} sessionId={sessionId} isSelected={(path) => selected.has(path)} onToggle={toggle} onPreview={setPreviewPath} />;
  } else if (startError || status === "error") {
    content = (
      <div className="flex h-full flex-col items-center justify-center gap-3 px-6 py-10 text-center" data-testid="host-picker-error">
        <AlertTriangle className="h-6 w-6 text-wc-warning" aria-hidden />
        <p className="max-w-sm break-words text-sm text-wc-text-muted">{startError ?? error}</p>
        <button type="button" onClick={() => { void openTab(tab === "home" ? "home" : "session"); }} className="inline-flex min-h-11 items-center gap-1.5 rounded-lg border border-wc-default px-3 text-sm text-wc-text-secondary">
          <RotateCw className="h-4 w-4" aria-hidden />{t(strings.composer.hostPickerRetry)}
        </button>
      </div>
    );
  } else if (loading || !model || model.kind !== "directory") {
    content = (
      <div className="flex h-full items-center justify-center gap-2 text-wc-text-muted">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
        <span>{t(strings.messagesFileViewer.loadingPreview)}</span>
      </div>
    );
  } else {
    content = (
      <DirectoryPreview
        model={model}
        text={null}
        listing={listing}
        listingRefreshing={listingRefreshing}
        onError={reportError}
        onNavigate={navigateTo}
        onLoadMore={loadMore}
        onListOptionsChange={setListOptions}
        loadingMore={loadingMore}
        selection={selection}
        imageSessionId={sessionId}
      />
    );
  }

  return (
    <FullPageDrawer
      open={open}
      onClose={onClose}
      avoidKeyboard
      title={t(strings.composer.hostPickerTitle)}
      closeLabel={t(strings.composer.hostPickerClose)}
      subheader={subheader}
      footer={footer}
      testId="host-file-picker"
    >
      <div className="flex h-full min-h-0 flex-col">{content}</div>
      <PickerPreview sessionId={sessionId} path={open ? previewPath : null} isSelected={(path) => selected.has(path)} onToggle={toggle} onClose={() => { setPreviewPath(null); }} />
    </FullPageDrawer>
  );
}

// PickerPreview shows one file in the regular file viewer, with a Select
// toggle in its header so the operator can pick what they just checked. It
// owns its own controller: walking around inside the preview never moves the
// picker's folder.
function PickerPreview({ sessionId, path, isSelected, onToggle, onClose }: { sessionId: string; path: string | null; isSelected: (path: string) => boolean; onToggle: (pick: HostAttachmentPick) => void; onClose: () => void }) {
  const { t } = useTranslation();
  const preview = useFilePreviewController(sessionId);
  const { openPreview, close } = preview;
  useEffect(() => {
    if (path) void openPreview(path);
    else close();
  }, [path, openPreview, close]);
  if (!path) return null;

  const model = preview.state.model;
  const target = model && !model.archiveMember && model.resolvedPath ? model : null;
  const checked = target ? isSelected(target.resolvedPath) : false;
  const action = target && (
    <button
      type="button"
      aria-pressed={checked}
      onClick={() => { onToggle({ path: target.resolvedPath, isDirectory: target.kind === "directory", sizeBytes: target.kind === "directory" ? 0 : target.sizeBytes }); }}
      data-testid="host-picker-preview-select"
      className={cn("inline-flex min-h-9 shrink-0 items-center gap-1.5 rounded-lg px-3 text-xs font-medium transition", checked ? "bg-wc-accent text-white" : "border border-wc-default bg-wc-surface-input text-wc-text-secondary hover:text-wc-text-primary")}
    >
      {checked && <Check className="h-3.5 w-3.5" aria-hidden />}
      {t(checked ? strings.composer.hostPickerPreviewSelected : strings.composer.hostPickerPreviewSelect)}
    </button>
  );
  return (
    <MessagesFileViewer
      state={preview.state}
      headerAction={action}
      onClose={onClose}
      onReopen={preview.reopen}
      onRendererError={preview.reportError}
      onNavigate={preview.navigateTo}
      onNavigateBack={preview.navigateBack}
      onLoadMore={preview.loadMore}
      onListOptionsChange={preview.setListOptions}
      onChooseCandidate={preview.chooseCandidate}
      onArchiveFolder={preview.navigateArchiveFolder}
      onArchiveEntry={preview.navigateArchiveEntry}
      onArchiveViewChange={preview.setArchiveView}
      imageSessionId={sessionId}
    />
  );
}

const TAB_LABEL = {
  session: strings.composer.hostPickerTabSession,
  home: strings.composer.hostPickerTabHome,
  mentioned: strings.composer.hostPickerTabMentioned,
} as const satisfies Record<PickerTab, string>;

const NO_EVENTS: readonly { text: string }[] = [];

// MentionedList offers the paths this session's messages already mention.
// Each is resolved on the host first, so a relative mention becomes the exact
// file it names, and one that matches nothing — or more than one place — is
// left out rather than guessed at.
function MentionedList({ open, sessionId, isSelected, onToggle, onPreview }: { open: boolean; sessionId: string; isSelected: (path: string) => boolean; onToggle: (pick: HostAttachmentPick) => void; onPreview: (path: string) => void }) {
  const { t } = useTranslation();
  const events = useConversationStore((s) => s.sessions[sessionId]?.events ?? NO_EVENTS);
  // Keyed by content: a new message that mentions nothing new must not
  // re-resolve the list or flash the loader.
  const mentionKey = useMemo(() => mentionedFilePaths(events.map((e) => e.text)).join("\n"), [events]);
  const [resolved, setResolved] = useState<PreviewModel[] | null>(null);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setResolved(null);
    const mentions = mentionKey ? mentionKey.split("\n") : [];
    void Promise.all(mentions.map((path) => resolveFilePreview(sessionId, path).catch(() => null))).then((models) => {
      if (cancelled) return;
      const seen = new Set<string>();
      setResolved(models.filter((m): m is PreviewModel => {
        if (!m || m.unresolved || !m.resolvedPath || (m.candidates?.length ?? 0) > 1 || seen.has(m.resolvedPath)) return false;
        seen.add(m.resolvedPath);
        return m.canPreview;
      }));
    });
    return () => { cancelled = true; };
  }, [open, mentionKey, sessionId]);

  if (!resolved) {
    return <div className="flex h-full items-center justify-center gap-2 text-wc-text-muted"><Loader2 className="h-4 w-4 animate-spin" aria-hidden /><span>{t(strings.composer.hostPickerMentionedLoading)}</span></div>;
  }
  if (resolved.length === 0) {
    return <p className="px-6 py-12 text-center text-sm text-wc-text-muted" data-testid="host-picker-mentioned-empty">{t(strings.composer.hostPickerMentionedEmpty)}</p>;
  }
  return (
    <ul className="min-h-0 flex-1 overflow-auto overscroll-contain" data-testid="host-picker-mentioned">
      {resolved.map((m) => {
        const isDirectory = m.kind === "directory";
        const checked = isSelected(m.resolvedPath);
        const name = basename(m.resolvedPath) || m.resolvedPath;
        return (
          <li key={m.resolvedPath} className={cn("flex items-stretch border-b border-wc-default", checked && "bg-wc-accent/10")}>
            <button
              type="button"
              role="checkbox"
              aria-checked={checked}
              aria-label={t(strings.composer.hostPickerToggle, { name })}
              onClick={() => { onToggle({ path: m.resolvedPath, isDirectory, sizeBytes: isDirectory ? 0 : m.sizeBytes }); }}
              data-testid="host-picker-mentioned-entry"
              data-selected={checked ? "true" : "false"}
              className="flex min-h-16 min-w-0 flex-1 items-center gap-3 px-3 py-3 text-start outline-none transition hover:bg-wc-surface-raised focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent"
            >
              <span aria-hidden="true" className={cn("flex h-5 w-5 shrink-0 items-center justify-center rounded border", checked ? "border-wc-accent bg-wc-accent text-white" : "border-wc-text-muted bg-wc-surface-input")}>{checked && <Check className="h-3.5 w-3.5" />}</span>
              <span aria-hidden="true" className={cn("relative flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg", isDirectory ? "bg-wc-accent/10 text-wc-accent" : "bg-wc-surface-raised text-wc-text-muted")}>
                {isDirectory ? <Folder className="h-4 w-4" /> : <FileText className="h-4 w-4" />}
                {canThumbnail(name) && m.previewId && <img src={previewThumbnailHref(sessionId, m.previewId, { size: 96 })} alt="" loading="lazy" decoding="async" draggable={false} onError={(e) => { e.currentTarget.hidden = true; }} className="absolute inset-0 h-full w-full object-cover" />}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium text-wc-text-primary" dir="auto">{name}</span>
                <span className="block truncate font-mono text-xs text-wc-text-muted" dir="auto">{m.resolvedPath}</span>
              </span>
            </button>
            {!isDirectory && (
              <button type="button" onClick={() => { onPreview(m.resolvedPath); }} aria-label={t(strings.composer.hostPickerPreview, { name })} title={t(strings.composer.hostPickerPreview, { name })} data-testid="host-picker-mentioned-preview"
                className="flex w-12 shrink-0 items-center justify-center text-wc-text-muted outline-none transition hover:bg-wc-surface-raised hover:text-wc-text-primary focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent">
                <ScanEye className="h-5 w-5" aria-hidden />
              </button>
            )}
          </li>
        );
      })}
    </ul>
  );
}
