import { Plus, Save } from "lucide-react";
import { Button } from "../../ui/button";
import { Textarea } from "../../ui/textarea";
import { selectors } from "../../../consts/selectors";
import { parseAcceptanceCriteria } from "./utils";

const COMMON_CRITERIA: ReadonlyArray<string> = [
  "All tests pass",
  "No new lint errors",
  "No TODOs introduced",
  "Docs updated",
  "Coverage ≥ 80%",
];

export interface AcceptanceCriteriaEditorProps {
  /** Current textarea value (may include in-flight edits not yet saved). */
  value: string;
  /** The saved canonical list. Save button hides when parsed `value` matches this. */
  saved: string[];
  isPending: boolean;
  onChange: (value: string) => void;
  onSave: () => void;
}

function appendIfMissing(text: string, line: string): string {
  const parsed = parseAcceptanceCriteria(text);
  const lower = parsed.map((entry) => entry.toLowerCase());
  if (lower.includes(line.toLowerCase())) return text;
  if (text.length === 0) return line;
  return text.endsWith("\n") ? `${text}${line}` : `${text}\n${line}`;
}

function listsEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i += 1) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}

export function AcceptanceCriteriaEditor({
  value,
  saved,
  isPending,
  onChange,
  onSave,
}: AcceptanceCriteriaEditorProps) {
  const parsed = parseAcceptanceCriteria(value);
  const isDirty = !listsEqual(parsed, saved);

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <span
          className="rounded-full border border-slate-700/80 bg-slate-900/60 px-2 py-0.5 text-[11px] text-slate-300"
          data-testid={selectors.initiativeDetails.criteriaCount}
        >
          {parsed.length} criterion{parsed.length === 1 ? "" : "s"}
        </span>
      </div>

      <div className="flex flex-wrap gap-1.5">
        {COMMON_CRITERIA.map((criterion) => (
          <button
            key={criterion}
            type="button"
            onClick={() => onChange(appendIfMissing(value, criterion))}
            disabled={isPending}
            data-testid={selectors.initiativeDetails.criteriaCommonChip}
            className="flex items-center gap-1 rounded-full border border-slate-700 bg-slate-800/60 px-2 py-0.5 text-[11px] text-slate-300 transition-colors hover:border-slate-500 hover:text-slate-100 disabled:cursor-not-allowed disabled:opacity-40"
          >
            <Plus className="h-3 w-3" aria-hidden="true" />
            {criterion}
          </button>
        ))}
      </div>

      <Textarea
        className="min-h-28"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={"All tests pass\nDocs updated"}
        data-testid={selectors.initiativeDetails.criteriaInput}
      />

      <div
        className="rounded-lg border border-slate-800 bg-slate-950/40 p-3"
        data-testid={selectors.initiativeDetails.criteriaPreview}
      >
        {parsed.length === 0 ? (
          <p className="text-xs italic text-slate-500">Enter one criterion per line.</p>
        ) : (
          <ol className="list-decimal space-y-0.5 pl-5 text-sm text-slate-300">
            {parsed.map((line, i) => (
              <li key={`${i}-${line}`}>{line}</li>
            ))}
          </ol>
        )}
      </div>

      {isDirty && (
        <div className="flex justify-end">
          <Button
            variant="outline"
            size="sm"
            onClick={onSave}
            disabled={isPending}
            data-testid={selectors.initiativeDetails.criteriaSave}
          >
            <Save className="mr-1.5 h-4 w-4" />
            {isPending ? "Saving..." : "Save Criteria"}
          </Button>
        </div>
      )}
    </div>
  );
}
