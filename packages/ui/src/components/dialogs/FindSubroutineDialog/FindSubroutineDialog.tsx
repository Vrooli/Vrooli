import { OwnerShape, RoutineVersion, uuidValidate, VisibilityType } from "@local/shared";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog.js";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { FindSubroutineDialogProps } from "../types.js";

const limitTo = [
    "RoutineSingleStep",
    "RoutineMultiStep",
] as const;

export function FindSubroutineDialog({
    handleComplete,
    nodeId,
    routineVersionId,
    ...params
}: FindSubroutineDialogProps) {
    const [ownerField] = useField<OwnerShape | null | undefined>("root.owner");

    const onComplete = useCallback((item: object) => {
        handleComplete(nodeId, item as RoutineVersion);
    }, [handleComplete, nodeId]);

    /**
     * Query conditions change depending on a few factors
     */
    const where = useMemo<{ [key: string]: object }>(() => {
        // If no routineVersionId, then we are creating a new routine
        if (!routineVersionId || !uuidValidate(routineVersionId)) return { visibility: VisibilityType.Public };
        return {
            // Ignore current routine
            excludeIds: [routineVersionId],
            // Don't include incomplete/internal routines, unless they're your own
            ...((ownerField as any)?.__typename === "User" ? {
                isCompleteWithRootExcludeOwnedByUserId: ownerField!.value!.id,
                isExternalWithRootExcludeOwnedByUserId: ownerField!.value!.id,
            } : {}),
            ...((ownerField as any)?.__typename === "Team" ? {
                isCompleteWithRootExcludeOwnedByTeamId: ownerField!.value!.id,
                isExternalWithRootExcludeOwnedByTeamId: ownerField!.value!.id,
            } : {}),
            ...(!ownerField ? {
                isCompleteWithRoot: true,
                isInternalWithRoot: false,
            } : {}),
            visibility: VisibilityType.OwnOrPublic,
        } as any;
    }, [ownerField, routineVersionId]);

    return <FindObjectDialog
        {...params}
        find="Full"
        handleComplete={onComplete}
        limitTo={limitTo}
        onlyVersioned
        where={where}
    />;
}
