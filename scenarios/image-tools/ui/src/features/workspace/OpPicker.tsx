import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";
import type { OperationInfo } from "../../api/ops";
import { opPresentation } from "./opCatalog";

export interface OpPickerProps {
  operations: OperationInfo[];
  operation: string;
  onSelect: (operation: string) => void;
}

/**
 * The humanized operation picker: a compact 2-column radiogroup of icon + label
 * tiles that replaces the raw `<select>`. The selected op's one-line
 * description renders beneath. Reuses the `operationSelect` test id on the
 * container so the smoke's "operation picker visible" check still resolves.
 */
export function OpPicker({ operations, operation, onSelect }: OpPickerProps) {
  const { t } = useTranslation();
  const selected = opPresentation(operation);

  return (
    <div className="flex flex-col gap-2">
      <span className="text-xs text-app-muted-foreground">
        {t(strings.workspace.operationLabelGroup)}
      </span>
      <div
        role="radiogroup"
        aria-label={t(strings.workspace.operationLabelGroup)}
        data-testid={selectors.workspace.operationSelect}
        className="grid grid-cols-2 gap-1"
      >
        {operations.map((op) => {
          const presentation = opPresentation(op.name);
          if (!presentation) {
            return null;
          }
          const { Icon, labelKey } = presentation;
          const active = op.name === operation;
          return (
            <button
              key={op.name}
              type="button"
              role="radio"
              aria-checked={active}
              data-testid={selectors.workspace.opOption({ name: op.name })}
              onClick={() => onSelect(op.name)}
              className={cn(
                "flex items-center gap-2 rounded-control border px-2.5 py-2 text-left text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
                active
                  ? "border-app-primary bg-app-surface-muted text-app-foreground"
                  : "border-app-border text-app-muted-foreground hover:text-app-foreground",
              )}
            >
              <Icon aria-hidden="true" className="h-4 w-4 shrink-0" />
              <span className="truncate">{t(labelKey)}</span>
            </button>
          );
        })}
      </div>
      {selected ? (
        <p
          data-testid={selectors.workspace.opDescription}
          className="text-xs text-app-muted-foreground"
        >
          {t(selected.descKey)}
        </p>
      ) : null}
    </div>
  );
}
