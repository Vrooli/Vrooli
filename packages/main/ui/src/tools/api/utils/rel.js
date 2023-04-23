import { exists } from "@local/utils";
import pkg from "lodash";
import { findSelection } from "./findSelection";
import { partialShape } from "./partialShape";
import { removeValuesUsingDot } from "./removeValuesUsingDot";
const { merge } = pkg;
export const rel = async (partial, selectionType, exceptions) => {
    const hasExceptions = exists(exceptions) && exists(exceptions.omit);
    const actualSelectionType = findSelection(partial, selectionType);
    let selectionData = { ...partial[actualSelectionType] };
    hasExceptions && removeValuesUsingDot(selectionData, ...(Array.isArray(exceptions.omit) ? exceptions.omit : [exceptions.omit]));
    selectionData = await partialShape(selectionData);
    if (["full", "list"].includes(actualSelectionType) && exists(partial.common)) {
        let commonData = partial.common;
        hasExceptions && removeValuesUsingDot(commonData, ...(Array.isArray(exceptions.omit) ? exceptions.omit : [exceptions.omit]));
        commonData = await partialShape(commonData);
        selectionData = merge(selectionData, commonData);
    }
    return { __typename: partial.__typename, __selectionType: actualSelectionType, ...selectionData };
};
//# sourceMappingURL=rel.js.map