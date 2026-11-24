import type { ReactNode } from "react";

import { Dialog } from "./ui/dialog";
import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { cn } from "../lib/utils";

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
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            disabled={busy}
            shape="square"
            className={cn("border border-app-border text-app-foreground hover:bg-app-surface-muted")}
          >
            {t(strings.common.cancel)}
          </Button>
          <Button
            size="sm"
            onClick={onConfirm}
            disabled={busy}
            data-testid={confirmTestId}
            className={danger ? "bg-app-danger text-white hover:brightness-95" : undefined}
            shape="square"
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
