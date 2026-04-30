import { FileText, Save } from "lucide-react";
import { Button } from "../../ui/button";
import { Textarea } from "../../ui/textarea";
import { selectors } from "../../../consts/selectors";

export function AcceptanceCriteriaEditor({
  value,
  isPending,
  onChange,
  onSave,
}: {
  value: string;
  isPending: boolean;
  onChange: (value: string) => void;
  onSave: () => void;
}) {
  return (
    <div className="rounded-lg border border-slate-800/80 bg-slate-900/55 p-4">
      <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
        <FileText className="h-3.5 w-3.5" />
        Acceptance Criteria
      </div>
      <Textarea
        className="mt-3 min-h-28"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder="One acceptance criterion per line."
        data-testid={selectors.initiativeDetails.criteriaInput}
      />
      <div className="mt-3 flex justify-end">
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
    </div>
  );
}
