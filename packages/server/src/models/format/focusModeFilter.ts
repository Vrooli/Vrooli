import { FocusModeFilterModelLogic } from "../base/types";
import { Formatter } from "../types";

export const FocusModeFilterFormat: Formatter<FocusModeFilterModelLogic> = {
    gqlRelMap: {
        __typename: "FocusModeFilter",
        focusMode: "FocusMode",
        tag: "Tag",
    },
    prismaRelMap: {
        __typename: "FocusModeFilter",
        focusMode: "FocusMode",
        tag: "Tag",
    },
    countFields: {},
};
