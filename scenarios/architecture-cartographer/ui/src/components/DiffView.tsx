import { cn } from "../lib/utils";
import { selectors } from "../consts/selectors";

export type DiffLineKind = "added" | "removed" | "context";

export interface DiffLine {
  kind: DiffLineKind;
  text: string;
  /** Optional line number for context. Renders as a fixed-width gutter. */
  lineNumber?: number;
}

export interface DiffViewProps {
  lines: ReadonlyArray<DiffLine>;
  /** Pre-translated labels — never rely on color alone. */
  addedLabel: string;
  removedLabel: string;
  className?: string;
}

const LINE_PREFIX: Record<DiffLineKind, string> = {
  added: "+",
  removed: "-",
  context: " ",
};

const LINE_CLASS: Record<DiffLineKind, string> = {
  added: "bg-app-success/10 text-app-success",
  removed: "bg-app-danger/10 text-app-danger",
  context: "text-app-foreground",
};

export function DiffView({ lines, addedLabel, removedLabel, className }: DiffViewProps) {
  return (
    <pre
      data-testid={selectors.shared.diffView.root}
      aria-label={`${addedLabel} / ${removedLabel}`}
      className={cn(
        "w-full overflow-x-auto rounded-panel border border-app-border bg-app-surface-muted p-3 font-mono text-xs leading-5",
        className,
      )}
    >
      {lines.map((line, i) => (
        <div
          key={i}
          className={cn("flex gap-3", LINE_CLASS[line.kind])}
          aria-label={
            line.kind === "added" ? addedLabel : line.kind === "removed" ? removedLabel : undefined
          }
        >
          <span aria-hidden="true" className="select-none text-app-muted-foreground w-8 text-right">
            {line.lineNumber ?? ""}
          </span>
          <span aria-hidden="true" className="select-none w-3">
            {LINE_PREFIX[line.kind]}
          </span>
          <span className="whitespace-pre-wrap break-all">{line.text}</span>
        </div>
      ))}
    </pre>
  );
}
