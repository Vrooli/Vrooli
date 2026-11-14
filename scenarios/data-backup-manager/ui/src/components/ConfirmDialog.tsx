import type { ReactNode } from "react";

import { Dialog } from "./ui/dialog";
import { Button } from "./ui/button";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * Confirmation dialog for consequential actions (deregister, delete, restore).
 * The confirm button can be tinted danger, and extra controls (e.g. the
 * "also delete repository" toggle, the restore location field) render in
 * `children` between the body and the actions.
 */
export function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title,
  body,
  confirmLabel,
  danger,
  busy,
  confirmTestId,
  children,
  "data-testid": testId,
}: {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  body?: string;
  confirmLabel: string;
  danger?: boolean;
  busy?: boolean;
  confirmTestId?: string;
  children?: ReactNode;
  "data-testid"?: string;
}) {
  const { t } = useTranslation();
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      data-testid={testId}
      footer={
        <>
          <Button variant="outline" size="sm" onClick={onClose} disabled={busy}>
            {t(strings.common.cancel)}
          </Button>
          <Button
            size="sm"
            onClick={onConfirm}
            disabled={busy}
            data-testid={confirmTestId}
            className={danger ? "bg-app-danger text-white hover:brightness-95" : undefined}
          >
            {confirmLabel}
          </Button>
        </>
      }
    >
      {body && <p className="text-sm text-app-muted-foreground">{body}</p>}
      {children}
    </Dialog>
  );
}
