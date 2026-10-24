import * as React from "react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { Dialog } from "../../components/ui/dialog";
import { Button } from "../../components/ui/button";
import { Textarea } from "../../components/ui/textarea";

export interface ApplyConfirmDialogProps {
  open: boolean;
  /** Set when force-apply is required (baseline was red before apply). */
  requiresNote: boolean;
  onClose: () => void;
  onConfirm: (args: { note: string }) => void;
  submitting?: boolean;
}

export function ApplyConfirmDialog({
  open,
  requiresNote,
  onClose,
  onConfirm,
  submitting = false,
}: ApplyConfirmDialogProps) {
  const { t } = useTranslation();
  const [note, setNote] = React.useState("");
  const [touched, setTouched] = React.useState(false);

  const noteError = requiresNote && touched && note.trim().length === 0;
  const canConfirm = !requiresNote || note.trim().length > 0;

  const handleConfirm = () => {
    setTouched(true);
    if (!canConfirm) return;
    onConfirm({ note: note.trim() });
  };

  React.useEffect(() => {
    if (!open) {
      setNote("");
      setTouched(false);
    }
  }, [open]);

  return (
    <Dialog
      open={open}
      onClose={onClose}
      ariaLabel={t(strings.pages.targetApply.forceConfirmTitle)}
    >
      <div
        data-testid={selectors.features.apply.confirmDialog.root}
        className="flex flex-col gap-3"
      >
        <h3 className="text-lg font-semibold">
          {t(strings.pages.targetApply.forceConfirmTitle)}
        </h3>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.targetApply.forceConfirmMessage)}
        </p>
        {requiresNote ? (
          <label className="flex flex-col gap-1 text-sm">
            <span>{t(strings.pages.targetApply.forceNoteLabel)}</span>
            <Textarea
              data-testid={selectors.features.apply.confirmDialog.noteInput}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder={t(strings.pages.targetApply.forceNotePlaceholder)}
              aria-invalid={noteError || undefined}
              aria-describedby={
                noteError ? selectors.features.apply.confirmDialog.noteError : undefined
              }
              rows={3}
            />
            {noteError ? (
              <span
                id={selectors.features.apply.confirmDialog.noteError}
                data-testid={selectors.features.apply.confirmDialog.noteError}
                className="text-app-danger"
              >
                {t(strings.pages.targetApply.forceNoteRequired)}
              </span>
            ) : null}
          </label>
        ) : null}
        <div className="flex justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            data-testid={selectors.features.apply.confirmDialog.cancelButton}
            onClick={onClose}
            disabled={submitting}
          >
            {t(strings.pages.targetApply.cancelButton)}
          </Button>
          <Button
            type="button"
            variant="default"
            data-testid={selectors.features.apply.confirmDialog.confirmButton}
            onClick={handleConfirm}
            disabled={submitting || (requiresNote && note.trim().length === 0)}
          >
            {submitting
              ? t(strings.pages.targetApply.applying)
              : t(strings.pages.targetApply.confirmButton)}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
