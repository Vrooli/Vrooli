import { FocusModeFilterModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "FocusModeFilter" as const;
export const FocusModeFilterFormat: Formatter<FocusModeFilterModelLogic> = {
    gqlRelMap: {
        __typename,
        focusMode: "FocusMode",
        tag: "Tag",
    },
    prismaRelMap: {
        __typename,
        focusMode: "FocusMode",
        tag: "Tag",
    },
    countFields: {},
};
