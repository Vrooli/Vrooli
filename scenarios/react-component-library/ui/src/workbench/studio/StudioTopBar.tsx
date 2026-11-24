import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  ChevronRight,
  Copy,
  FileUp,
  History,
  Monitor,
  Play,
  Plus,
  Redo2,
  Share2,
  Trash2,
  Smartphone,
  Tablet,
  Undo2,
  X,
} from "lucide-react";

import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import { formatRelative } from "../data";
import type { DesignDocument, DesignDevice } from "../design-types";
import type { CommandStack } from "./history";
import { Segmented, WbButton } from "../primitives";
import { cx } from "../cx";

export function StudioTopBar({
  designs,
  openIds,
  activeId,
  onSelect,
  onClose,
  onNew,
  onImportHtml,
  onOpen,
  onDuplicate,
  onDelete,
  canUndo,
  canRedo,
  onUndo,
  onRedo,
  history,
  onJump,
  device,
  onDevice,
  onShare,
  onPresent,
  dirty,
}: {
  designs: DesignDocument[];
  openIds: string[];
  activeId: string;
  onSelect: (id: string) => void;
  onClose: (id: string) => void;
  onNew: () => void;
  /** Opens the HTML import dialog. Optional so the bar stays reusable. */
  onImportHtml?: () => void;
  onOpen: (id: string) => void;
  onDuplicate: () => void;
  onDelete: () => void;
  canUndo: boolean;
  canRedo: boolean;
  onUndo: () => void;
  onRedo: () => void;
  history: CommandStack<DesignDocument>;
  onJump: (position: number) => void;
  device: DesignDevice;
  onDevice: (device: DesignDevice) => void;
  onShare: () => void;
  onPresent: () => void;
  dirty: boolean;
}) {
  const { t } = useTranslation();
  const [menuOpen, setMenuOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  useEffect(() => {
    if (!confirmDelete) return;
    const timer = window.setTimeout(() => setConfirmDelete(false), 4000);
    return () => window.clearTimeout(timer);
  }, [confirmDelete]);
  const open = designs.filter((design) => openIds.includes(design.id));
  const closed = designs.filter((design) => !openIds.includes(design.id));
  const active = designs.find((design) => design.id === activeId);

  return (
    <>
      <header className="studio-topbar">
        <div className="studio-crumbs">
          <Link to={appRoutes.design}>{t("workbench.studio.designs", { defaultValue: "Designs" })}</Link>
          <ChevronRight aria-hidden size={12} />
          <b>{active?.name ?? t("workbench.studio.untitled", { defaultValue: "Untitled design" })}</b>
        </div>

        <div className="studio-tabs" role="tablist" aria-label={t("workbench.studio.tabs", { defaultValue: "Open designs" })}>
          {open.map((design) => {
            const current = design.id === activeId;
            return (
              <span key={design.id} className={cx("studio-tab", current && "is-active")}>
                {current && dirty ? <span className="studio-tab-dot" aria-hidden /> : null}
                <button
                  type="button"
                  role="tab"
                  aria-selected={current}
                  className="studio-tab-name"
                  onClick={() => onSelect(design.id)}
                >
                  <span>{design.name}</span>
                  <small>
                    {t("workbench.studio.revisionShort", {
                      defaultValue: "v{{revision}}",
                      revision: design.revision,
                    })}
                  </small>
                </button>
                {open.length > 1 ? (
                  <button
                    type="button"
                    className="studio-tab-close"
                    aria-label={t("workbench.studio.closeDesign", {
                      defaultValue: "Close {{name}}",
                      name: design.name,
                    })}
                    onClick={() => onClose(design.id)}
                  >
                    <X aria-hidden size={11} />
                  </button>
                ) : null}
              </span>
            );
          })}
          <button
            type="button"
            className="studio-tab-add"
            aria-label={t("workbench.studio.openDesign", { defaultValue: "Open a design" })}
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen((value) => !value)}
          >
            <Plus aria-hidden size={13} />
          </button>
        </div>

        <div className="studio-toolgroup">
          <button
            type="button"
            className="studio-iconbtn"
            aria-label={t("workbench.studio.undo", { defaultValue: "Undo" })}
            disabled={!canUndo}
            onClick={onUndo}
          >
            <Undo2 aria-hidden size={15} />
          </button>
          <button
            type="button"
            className="studio-iconbtn"
            aria-label={t("workbench.studio.redo", { defaultValue: "Redo" })}
            disabled={!canRedo}
            onClick={onRedo}
          >
            <Redo2 aria-hidden size={15} />
          </button>
          <button
            type="button"
            className={cx("studio-iconbtn", historyOpen && "is-active")}
            aria-label={t("workbench.studio.historyOpen", { defaultValue: "History" })}
            aria-expanded={historyOpen}
            onClick={() => setHistoryOpen((value) => !value)}
          >
            <History aria-hidden size={15} />
          </button>
          <button
            type="button"
            className="studio-iconbtn"
            aria-label={t("workbench.studio.duplicateDesign", {
              defaultValue: "Duplicate this design",
            })}
            onClick={onDuplicate}
          >
            <Copy aria-hidden size={14} />
          </button>
          <button
            type="button"
            className={cx("studio-iconbtn", confirmDelete && "is-danger")}
            aria-label={
              confirmDelete
                ? t("workbench.studio.confirmDelete", { defaultValue: "Confirm delete design" })
                : t("workbench.studio.deleteDesign", { defaultValue: "Delete this design" })
            }
            disabled={designs.length <= 1}
            onClick={() => {
              if (!confirmDelete) {
                setConfirmDelete(true);
                return;
              }
              setConfirmDelete(false);
              onDelete();
            }}
          >
            <Trash2 aria-hidden size={14} />
          </button>
          {confirmDelete ? (
            <span className="studio-confirm-hint" role="status">
              {t("workbench.studio.deleteConfirmHint", {
                defaultValue: "Click again to delete. This cannot be undone.",
              })}
            </span>
          ) : null}
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

        <div className="studio-top-actions">
          <WbButton variant="ghost" size="sm" onClick={onShare}>
            <Share2 aria-hidden size={13} />
            <span className="studio-label-full">
              {t("workbench.studio.share", { defaultValue: "Share" })}
            </span>
          </WbButton>
          <WbButton variant="primary" size="sm" onClick={onPresent}>
            <Play aria-hidden size={13} />
            <span className="studio-label-full">
              {t("workbench.studio.present", { defaultValue: "Present" })}
            </span>
          </WbButton>
        </div>
      </header>

      {menuOpen ? (
        <>
          <button
            type="button"
            className="studio-dismiss"
            aria-label={t("workbench.studio.closeMenu", { defaultValue: "Close menu" })}
            onClick={() => setMenuOpen(false)}
          />
          <div
            className="studio-menu"
            role="dialog"
            aria-label={t("workbench.studio.openDesign", { defaultValue: "Open a design" })}
          >
            <span className="wb-caption">
              {t("workbench.studio.newDesign", { defaultValue: "New design" })}
            </span>
            <button
              type="button"
              className="studio-menu-row"
              onClick={() => {
                setMenuOpen(false);
                onNew();
              }}
            >
              <Plus aria-hidden size={13} />
              {t("workbench.studio.blankDesign", { defaultValue: "Blank design" })}
            </button>
            {onImportHtml ? (
              <button
                type="button"
                className="studio-menu-row"
                onClick={() => {
                  setMenuOpen(false);
                  onImportHtml();
                }}
              >
                <FileUp aria-hidden size={13} />
                {t("workbench.studio.importHtmlDesign", {
                  defaultValue: "Import an HTML page",
                })}
              </button>
            ) : null}
            {closed.length > 0 ? (
              <>
                <span className="wb-caption studio-menu-label">
                  {t("workbench.studio.closedDesigns", { defaultValue: "Saved designs" })}
                </span>
                {closed.map((design) => (
                  <button
                    key={design.id}
                    type="button"
                    className="studio-menu-row"
                    onClick={() => {
                      setMenuOpen(false);
                      onOpen(design.id);
                    }}
                  >
                    <Monitor aria-hidden size={13} />
                    <span>{design.name}</span>
                    <small>{formatRelative(new Date(design.updatedAt))}</small>
                  </button>
                ))}
              </>
            ) : null}
          </div>
        </>
      ) : null}

      {historyOpen ? (
        <>
          <button
            type="button"
            className="studio-dismiss"
            aria-label={t("workbench.studio.closeMenu", { defaultValue: "Close menu" })}
            onClick={() => setHistoryOpen(false)}
          />
          <div
            className="studio-menu studio-history"
            role="dialog"
            aria-label={t("workbench.studio.history", { defaultValue: "History" })}
          >
            <span className="wb-caption">
              {t("workbench.studio.history", { defaultValue: "History" })}
            </span>
            <button
              type="button"
              className={cx("studio-history-row", history.index === 0 && "is-current")}
              onClick={() => {
                onJump(0);
                setHistoryOpen(false);
              }}
            >
              <span className="studio-history-index">0</span>
              <span>{t("workbench.studio.historyStart", { defaultValue: "Starting point" })}</span>
            </button>
            {[...history.entries].reverse().map((entry, reverseIndex) => {
              const position = history.entries.length - reverseIndex;
              return (
                <button
                  key={entry.id}
                  type="button"
                  className={cx("studio-history-row", history.index === position && "is-current")}
                  onClick={() => {
                    onJump(position);
                    setHistoryOpen(false);
                  }}
                >
                  <span className="studio-history-index">{position}</span>
                  <span>{entry.label}</span>
                  <small>{formatRelative(new Date(entry.at))}</small>
                </button>
              );
            })}
            {history.entries.length === 0 ? (
              <p className="studio-menu-empty">
                {t("workbench.studio.historyEmpty", {
                  defaultValue: "Edits appear here as undoable steps.",
                })}
              </p>
            ) : null}
          </div>
        </>
      ) : null}
    </>
  );
}
