import { Card } from "../ui/card";
import { selectors } from "../../consts/selectors";

export function WorkshopTab() {
  return (
    <div className="space-y-6">
      <Card data-testid={selectors.settings.workshopSettings}>
        <div className="flex items-center gap-2">
          <h3 className="text-lg font-medium text-slate-200">Plan Workshop</h3>
        </div>
        <p className="mt-1 text-sm text-slate-400">Start a review explicitly from a backlog item. Plan Workshop sessions keep review, reconciliation, and candidate acceptance together; there are no automatic rounds or readiness controls to configure.</p>
      </Card>
    </div>
  );
}
