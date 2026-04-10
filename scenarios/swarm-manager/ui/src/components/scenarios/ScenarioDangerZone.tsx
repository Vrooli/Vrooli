import { Loader2, Trash2 } from "lucide-react";
import { Button } from "../ui/button";
import { DetailSection } from "../detail/DetailSection";
import { selectors } from "../../consts/selectors";

export interface ScenarioDangerZoneProps {
  onDeleteClick: () => void;
  deletePending: boolean;
  deleteError: boolean;
}

export function ScenarioDangerZone({ onDeleteClick, deletePending, deleteError }: ScenarioDangerZoneProps) {
  return (
    <DetailSection title="Danger Zone" className="text-red-300">
      <div className="flex items-center justify-between rounded-lg bg-slate-700/30 p-4">
        <div className="space-y-1">
          <span className="font-medium text-slate-200">Delete Scenario</span>
          <p className="text-sm text-slate-400">
            Permanently remove this scenario from the catalog. This action cannot be undone.
          </p>
        </div>
        <Button
          variant="destructive"
          size="sm"
          onClick={onDeleteClick}
          disabled={deletePending}
          data-testid={selectors.scenarioDetails.deleteButton}
        >
          {deletePending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Deleting...
            </>
          ) : (
            <>
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </>
          )}
        </Button>
      </div>

      {deleteError && (
        <div className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-400">
          Failed to delete scenario. Please try again.
        </div>
      )}
    </DetailSection>
  );
}
