import { OwnerShape, RoutineVersion, uuidValidate, VisibilityType } from "@local/shared";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { FindSubroutineDialogProps } from "../types";

export const FindSubroutineDialog = ({
    handleComplete,
    nodeId,
    routineVersionId,
    ...params
}: FindSubroutineDialogProps) => {
    const [ownerField] = useField<OwnerShape | null | undefined>("root.owner");

    const onComplete = useCallback((item: any) => {
        handleComplete(nodeId, item as RoutineVersion);
    }, [handleComplete, nodeId]);

    /**
     * Query conditions change depending on a few factors
     */
    const where = useMemo<{ [key: string]: object }>(() => {
        // If no routineVersionId, then we are creating a new routine
        if (!routineVersionId || !uuidValidate(routineVersionId)) return { visibility: VisibilityType.All };
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
            visibility: VisibilityType.All,
        } as any;
    }, [ownerField, routineVersionId]);

    return <FindObjectDialog
        {...params}
        find="Full"
        handleComplete={onComplete}
        limitTo={["Routine"]}
        onlyVersioned
        where={where}
    />;
};
