import { RoutineVersion, uuidValidate, VisibilityType } from "@local/shared";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { OwnerShape } from "utils/shape/models/types";
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
    const where = useMemo(() => {
        // If no routineVersionId, then we are creating a new routine
        if (!routineVersionId || !uuidValidate(routineVersionId)) return { visibility: VisibilityType.All };
        return () => ({
            // Ignore current routine
            excludeIds: [routineVersionId],
            // Don't include incomplete/internal routines, unless they're your own
            ...((ownerField as any)?.__typename === "User" ? {
                isCompleteWithRootExcludeOwnedByUserId: ownerField!.value!.id,
                isExternalWithRootExcludeOwnedByUserId: ownerField!.value!.id,
            } : {}),
            ...((ownerField as any)?.__typename === "Organization" ? {
                isCompleteWithRootExcludeOwnedByOrganizationId: ownerField!.value!.id,
                isExternalWithRootExcludeOwnedByOrganizationId: ownerField!.value!.id,
            } : {}),
            ...(!ownerField ? {
                isCompleteWithRoot: true,
                isInternalWithRoot: false,
            } : {}),
            visibility: VisibilityType.All,
        });
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
