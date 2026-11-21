import { useState } from "react";
import { Link } from "react-router-dom";
import {
  ChevronRight,
  Copy,
  FileUp,
  Monitor,
  Play,
  Plus,
  Redo2,
  Save,
  Share2,
  Smartphone,
  Tablet,
  Trash2,
  Undo2,
} from "lucide-react";

import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import type { DesignDevice } from "../designs";
import type { WorkspacePage } from "../../api/workspaces";
import { Segmented } from "../primitives";
import { cx } from "../cx";

/**
 * The workspace studio's chrome. It keeps the mature studio's top-bar language
 * (crumbs, tabs, undo/redo, device switch, present) while the tab strip selects
 * a page rather than a separate design document.
 */
export function WorkspaceTopBar({
  workspaceName,
  pages,
  activePageId,
  onSelectPage,
  onAddPage,
  onDuplicatePage,
  onImportHtml,
  onRemovePage,
  canUndo,
  canRedo,
  onUndo,
  onRedo,
  dirty,
  saving,
  saveFailed,
  onSave,
  device,
  onDevice,
  onPresent,
  onShare,
  readOnly = false,
}: {
  workspaceName: string;
  pages: WorkspacePage[];
  activePageId: string;
  onSelectPage: (id: string) => void;
  onAddPage: () => void;
  onDuplicatePage: () => void;
  /** Opens the HTML import dialog. Optional so the bar stays reusable. */
  onImportHtml?: () => void;
  onRemovePage: () => void;
  canUndo: boolean;
  canRedo: boolean;
  onUndo: () => void;
  onRedo: () => void;
  dirty: boolean;
  saving: boolean;
  saveFailed: boolean;
  onSave: () => void;
  device: DesignDevice;
  onDevice: (device: DesignDevice) => void;
  onPresent: () => void;
  onShare: () => void;
  /** A curated template previews; it cannot be saved or restructured. */
  readOnly?: boolean;
}) {
  const { t } = useTranslation();
  const [confirmRemove, setConfirmRemove] = useState(false);

  return (
    <header className="studio-topbar">
      <div className="studio-crumbs">
        <Link to={appRoutes.design}>{t("workbench.studio.workspaces", { defaultValue: "Workspaces" })}</Link>
        <ChevronRight aria-hidden size={12} />
        <b>{workspaceName}</b>
      </div>

      <div className="studio-tabs">
        <div
          className="studio-tablist"
          role="tablist"
          aria-label={t("workbench.workspace.pages", { defaultValue: "Pages" })}
        >
          {pages.map((page) => {
          const current = page.id === activePageId;
          return (
            <span key={page.id} className={cx("studio-tab", current && "is-active")}>
              {current && dirty ? <span className="studio-tab-dot" aria-hidden /> : null}
              <button
                type="button"
                role="tab"
                aria-selected={current}
                className="studio-tab-name"
                onClick={() => onSelectPage(page.id)}
              >
                <span>{page.name}</span>
                <small>{page.displays.length}</small>
              </button>
            </span>
            );
          })}
        </div>
        <button
          type="button"
          className="studio-tab-add"
          aria-label={t("workbench.workspace.addPage", { defaultValue: "Add page" })}
          disabled={readOnly}
          onClick={onAddPage}
        >
          <Plus aria-hidden size={13} />
        </button>
      </div>

      <div className="studio-toolgroup">
        <button
          type="button"
          className="studio-iconbtn"
          aria-label={t("workbench.studio.undo", { defaultValue: "Undo" })}
          disabled={readOnly || !canUndo}
          onClick={onUndo}
        >
          <Undo2 aria-hidden size={15} />
        </button>
        <button
          type="button"
          className="studio-iconbtn"
          aria-label={t("workbench.studio.redo", { defaultValue: "Redo" })}
          disabled={readOnly || !canRedo}
          onClick={onRedo}
        >
          <Redo2 aria-hidden size={15} />
        </button>
        <button
          type="button"
          className="studio-iconbtn"
          aria-label={t("workbench.workspace.duplicatePage", { defaultValue: "Duplicate page" })}
          onClick={onDuplicatePage}
          disabled={readOnly}
        >
          <Copy aria-hidden size={14} />
        </button>
        {onImportHtml ? (
          <button
            type="button"
            className="studio-iconbtn"
            aria-label={t("workbench.studio.importHtml", { defaultValue: "Import an HTML page" })}
            disabled={readOnly}
            onClick={onImportHtml}
          >
            <FileUp aria-hidden size={14} />
          </button>
        ) : null}
        <button
          type="button"
          className={cx("studio-iconbtn", confirmRemove && "is-danger")}
          aria-label={
            confirmRemove
              ? t("workbench.workspace.confirmRemovePage", { defaultValue: "Confirm remove page" })
              : t("workbench.workspace.removePage", { defaultValue: "Remove page" })
          }
          disabled={readOnly || pages.length <= 1}
          onClick={() => {
            if (!confirmRemove) {
              setConfirmRemove(true);
              globalThis.setTimeout(() => setConfirmRemove(false), 4000);
              return;
            }
            setConfirmRemove(false);
            onRemovePage();
          }}
        >
          <Trash2 aria-hidden size={14} />
        </button>
        <button
          type="button"
          className="studio-iconbtn"
          aria-label={t("workbench.workspace.save", { defaultValue: "Save workspace" })}
          disabled={readOnly || saving}
          onClick={onSave}
        >
          <Save aria-hidden size={14} />
        </button>
      </div>

      <Segmented
        label={t("workbench.studio.device", { defaultValue: "Device" })}
        value={device}
        onChange={onDevice}
        items={[
          {
            id: "desktop",
            label: <Monitor aria-hidden size={13} />,
            ariaLabel: t("workbench.studio.desktop", { defaultValue: "Desktop" }),
          },
          {
            id: "tablet",
            label: <Tablet aria-hidden size={13} />,
            ariaLabel: t("workbench.studio.tablet", { defaultValue: "Tablet" }),
          },
          {
            id: "phone",
            label: <Smartphone aria-hidden size={13} />,
            ariaLabel: t("workbench.studio.phone", { defaultValue: "Phone" }),
          },
        ]}
      />

      <div className="studio-topbar-end">
        <span
          className={cx(
            "studio-save-chip",
            readOnly
              ? "is-readonly"
              : saveFailed
                ? "is-danger"
                : dirty || saving
                  ? "is-pending"
                  : "is-saved",
          )}
          role="status"
        >
          {readOnly
            ? t("workbench.workspace.templateChip", { defaultValue: "Template" })
            : saveFailed
              ? t("workbench.studio.saveFailed", { defaultValue: "Save failed" })
              : saving
                ? t("workbench.studio.saving", { defaultValue: "Saving…" })
                : dirty
                  ? t("workbench.studio.draft", { defaultValue: "Draft" })
                  : t("workbench.studio.saved", { defaultValue: "Saved" })}
        </span>
        <button
          type="button"
          className="studio-iconbtn"
          aria-label={t("workbench.workspace.share", { defaultValue: "Share this workspace" })}
          onClick={onShare}
        >
          <Share2 aria-hidden size={14} />
        </button>
        <button
          type="button"
          className="studio-iconbtn"
          aria-label={t("workbench.studio.present", { defaultValue: "Present" })}
          onClick={onPresent}
        >
          <Play aria-hidden size={14} />
        </button>
      </div>
    </header>
  );
}
