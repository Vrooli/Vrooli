import { useState } from "react";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";

interface PullModelModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (modelName: string) => void;
  pending?: boolean;
}

export function PullModelModal({ open, onClose, onConfirm, pending }: PullModelModalProps) {
  const { t } = useTranslation();
  const [name, setName] = useState("");

  if (!open) return null;

  return (
    <div
      role="dialog"
      aria-label={t(strings.status.pullDialogLabel)}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
    >
      <div className="flex w-full max-w-md flex-col gap-3 rounded-control border border-app-border bg-app-surface p-4 shadow-xl">
        <h2 className="text-base font-semibold text-app-foreground">{t(strings.status.pullHeading)}</h2>
        <p className="text-xs text-app-muted-foreground">
          {t(strings.status.pullHint, {
            commandCode: "ollama pull",
            example1: "phi3:mini",
            example2: "llama3.1:8b",
          })}
        </p>
        <Input
          // autoFocus is intentional: this is a modal dialog opened by an
          // explicit user action; focusing the only input is the expected
          // behavior for keyboard users.
          // eslint-disable-next-line jsx-a11y/no-autofocus
          autoFocus
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t(strings.status.pullPlaceholder)}
          aria-label={t(strings.status.pullFieldLabel)}
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose} disabled={pending}>
            {t(strings.status.pullCancel)}
          </Button>
          <Button
            size="sm"
            onClick={() => onConfirm(name.trim())}
            disabled={pending || name.trim() === ""}
          >
            {pending ? t(strings.status.pullPulling) : t(strings.status.pullConfirm)}
          </Button>
        </div>
      </div>
    </div>
  );
}

export default PullModelModal;
