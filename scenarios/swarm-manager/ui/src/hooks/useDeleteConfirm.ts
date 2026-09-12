/**
 * useDeleteConfirm — the single reuse seam for entity deletion.
 *
 * Every delete site calls `requestDelete(...)` from its delete button and
 * renders `<ConfirmDialog {...dialogProps} />`. The configured confirmation
 * level for the entity type (none / simple / strong, from Settings) decides
 * the behavior — callers no longer hand-roll dialog shape:
 *
 *   - none   → invoke onConfirm immediately, no dialog
 *   - simple → open an OK/Cancel dialog
 *   - strong → open a type-to-confirm dialog (with a copy button). For bulk
 *              deletes the token is `DELETE <count>` since N names can't be typed.
 *
 * The hook tracks its own processing state while awaiting onConfirm and keeps
 * the dialog open if onConfirm rejects, so the user can retry or cancel.
 */

import { useCallback, useState, type ReactNode } from "react";
import { useRuntimeConfig } from "./useRuntimeConfig";
import { DELETABLE_ENTITY_BY_TYPE, type DeletableEntityType } from "../lib/deletable-entities";
import type { DeleteConfirmLevel } from "../types/settings";

export interface RequestDeleteArgs {
  /** Human name of the entity, shown in the prompt and typed for `strong`. */
  entityName: string;
  /** Called when the user confirms (or immediately when level is `none`). */
  onConfirm: () => void | Promise<void>;
  /** Number of entities for a bulk delete. Omit/1 for a single delete. */
  count?: number;
  /** Override the dialog title. */
  title?: string;
  /** Override the dialog body text. */
  description?: string;
  /** Override the confirm button label. */
  confirmLabel?: string;
  /** Optional side panel (e.g. scenario archive options). */
  sidePanel?: ReactNode;
  /** Optional checkbox forwarded to the dialog. */
  checkboxContent?: {
    label: string;
    checked: boolean;
    onChange: (checked: boolean) => void;
    testId?: string;
  };
  /** Optional per-site test IDs. */
  testIds?: {
    dialog?: string;
    confirmButton?: string;
    cancelButton?: string;
    copyButton?: string;
  };
}

/** Props ready to spread into `<ConfirmDialog />`. */
export interface DeleteConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  description: string;
  confirmationText?: string;
  confirmLabel?: string;
  isLoading: boolean;
  checkboxContent?: RequestDeleteArgs["checkboxContent"];
  sidePanel?: ReactNode;
  testIds?: RequestDeleteArgs["testIds"];
}

export interface UseDeleteConfirmResult {
  /** Trigger a delete; resolves the level and opens the dialog or acts now. */
  requestDelete: (args: RequestDeleteArgs) => void;
  /** Spread into `<ConfirmDialog />` at the call site. */
  dialogProps: DeleteConfirmDialogProps;
  /** The configured level for this entity type (for conditional UI copy). */
  level: DeleteConfirmLevel;
}

/**
 * The token a user types to confirm a `strong` delete. Single deletes use the
 * entity name; bulk deletes use a fixed `DELETE <count>` token (typing N
 * distinct names is impractical).
 */
export function strongConfirmToken(entityName: string, count?: number): string {
  return count && count > 1 ? `DELETE ${count}` : entityName;
}

function defaultTitle(entityType: DeletableEntityType, count?: number): string {
  const meta = DELETABLE_ENTITY_BY_TYPE[entityType];
  // Strip a trailing plural "s" for the singular case heuristically; fall back
  // to the registry label for bulk.
  if (count && count > 1) return `Delete ${count} ${meta.label}`;
  const singular = meta.label.replace(/s$/, "");
  return `Delete ${singular}`;
}

function defaultDescription(args: RequestDeleteArgs): string {
  const { entityName, count } = args;
  if (count && count > 1) {
    return `This permanently deletes ${count} items. This action cannot be undone.`;
  }
  return `This permanently deletes "${entityName}". This action cannot be undone.`;
}

const CLOSED_ARGS: RequestDeleteArgs = { entityName: "", onConfirm: () => {} };

export function useDeleteConfirm(entityType: DeletableEntityType): UseDeleteConfirmResult {
  const { getDeleteConfirmLevel } = useRuntimeConfig();
  const level = getDeleteConfirmLevel(entityType);

  const [isOpen, setIsOpen] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const [args, setArgs] = useState<RequestDeleteArgs>(CLOSED_ARGS);

  const close = useCallback(() => {
    setIsOpen(false);
    setIsProcessing(false);
  }, []);

  const runConfirm = useCallback(
    async (confirmArgs: RequestDeleteArgs, fromDialog: boolean) => {
      try {
        if (fromDialog) setIsProcessing(true);
        await confirmArgs.onConfirm();
        if (fromDialog) {
          setIsOpen(false);
          setIsProcessing(false);
        }
      } catch {
        // Keep the dialog open on failure so the user can retry/cancel.
        if (fromDialog) setIsProcessing(false);
      }
    },
    [],
  );

  const requestDelete = useCallback(
    (nextArgs: RequestDeleteArgs) => {
      // Re-read the level at click time so a settings change mid-session is
      // honored without remounting.
      const currentLevel = getDeleteConfirmLevel(entityType);
      if (currentLevel === "none") {
        void runConfirm(nextArgs, false);
        return;
      }
      setArgs(nextArgs);
      setIsProcessing(false);
      setIsOpen(true);
    },
    [entityType, getDeleteConfirmLevel, runConfirm],
  );

  const confirmationText =
    level === "strong" ? strongConfirmToken(args.entityName, args.count) : undefined;

  const dialogProps: DeleteConfirmDialogProps = {
    isOpen,
    onClose: close,
    onConfirm: () => {
      void runConfirm(args, true);
    },
    title: args.title ?? defaultTitle(entityType, args.count),
    description: args.description ?? defaultDescription(args),
    confirmationText,
    confirmLabel: args.confirmLabel ?? "Delete",
    isLoading: isProcessing,
    checkboxContent: args.checkboxContent,
    sidePanel: args.sidePanel,
    testIds: args.testIds,
  };

  return { requestDelete, dialogProps, level };
}
