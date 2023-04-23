import { jsx as _jsx } from "react/jsx-runtime";
import { VisibilityType } from "@local/consts";
import { uuidValidate } from "@local/uuid";
import { useField } from "formik";
import { useCallback, useMemo } from "react";
import { FindObjectDialog } from "../FindObjectDialog/FindObjectDialog";
export const FindSubroutineDialog = ({ handleComplete, nodeId, routineVersionId, ...params }) => {
    const [ownerField] = useField("root.owner");
    const onComplete = useCallback((item) => {
        handleComplete(nodeId, item);
    }, [handleComplete, nodeId]);
    const where = useMemo(() => {
        if (!routineVersionId || !uuidValidate(routineVersionId))
            return { visibility: VisibilityType.All };
        return {
            excludeIds: [routineVersionId],
            ...(ownerField?.__typename === "User" ? {
                isCompleteWithRootExcludeOwnedByUserId: ownerField.value.id,
                isExternalWithRootExcludeOwnedByUserId: ownerField.value.id,
            } : {}),
            ...(ownerField?.__typename === "Organization" ? {
                isCompleteWithRootExcludeOwnedByOrganizationId: ownerField.value.id,
                isExternalWithRootExcludeOwnedByOrganizationId: ownerField.value.id,
            } : {}),
            ...(!ownerField ? {
                isCompleteWithRoot: true,
                isInternalWithRoot: false,
            } : {}),
            visibility: VisibilityType.All,
        };
    }, [ownerField, routineVersionId]);
    return _jsx(FindObjectDialog, { ...params, find: "Full", handleComplete: onComplete, limitTo: ["RoutineVersion"], searchData: {
            searchType: "RoutineVersion",
            where,
        } });
};
//# sourceMappingURL=FindSubroutineDialog.js.map