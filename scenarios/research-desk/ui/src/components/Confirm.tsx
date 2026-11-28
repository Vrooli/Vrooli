import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { useId, useState, type ReactNode } from "react";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";

export interface ConfirmProps {
  open: boolean;
  title: string;
  /** One sentence: what happens, where. */
  description: string;
  /** The exact text that will be written or typed, and where it goes. */
  children?: ReactNode;
  confirmLabel: string;
  destructive?: boolean;
  /** When set, the operator must give a reason before confirming. */
  reasonLabel?: string;
  onConfirm: (reason: string) => Promise<unknown>;
  onClose: () => void;
}

/**
 * Every outward or destructive action passes through here: never one tap.
 * The dialog stays open on failure and shows the API's reason, so nothing
 * fails silently; it closes only after the action succeeded.
 */
export function Confirm({ open, title, description, children, confirmLabel, destructive, reasonLabel, onConfirm, onClose }: ConfirmProps) {
  const { t } = useTranslation();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const [reason, setReason] = useState("");
  const reasonId = useId();
  const close = () => {
    setError(undefined);
    setReason("");
    onClose();
  };
  const confirm = async () => {
    if (reasonLabel && !reason.trim()) {
      setError(t(strings.confirm.reasonRequired));
      return;
    }
    setBusy(true);
    setError(undefined);
    try {
      await onConfirm(reason.trim());
      close();
    } catch (err) {
      setError(errorMessage(err, t));
    } finally {
      setBusy(false);
    }
  };
  return (
    <AlertDialog
      open={open}
      title={title}
      description={description}
      destructive={destructive}
      busy={busy}
      status={error ? "error" : "default"}
      errorMessage={error}
      confirmLabel={confirmLabel}
      cancelLabel={t(strings.confirm.cancel)}
      busyLabel={t(strings.confirm.busy)}
      closeLabel={t(strings.confirm.cancel)}
      onConfirm={confirm}
      onCancel={close}
    >
      <div className="flex flex-col gap-space-sm">
        {children}
        {reasonLabel ? (
          <label htmlFor={reasonId} className="flex flex-col gap-space-2xs text-body-sm">
            <span className="font-medium">{reasonLabel}</span>
            <Textarea id={reasonId} value={reason} onChange={(e) => setReason(e.target.value)} rows={3} required />
          </label>
        ) : null}
      </div>
    </AlertDialog>
  );
}

/** The literal text an action types or writes, shown as the operator will see it. */
export function Exact({ label, text }: { label: string; text: string }) {
  return (
    <div className="flex flex-col gap-space-3xs">
      <span className="rd-exact-label text-caption text-app-muted-foreground">{label}</span>
      <pre className="rd-exact">{text}</pre>
    </div>
  );
}
