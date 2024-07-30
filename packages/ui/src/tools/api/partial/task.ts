import { AutoFillResult, CheckTaskStatusesResult } from "@local/shared";
import { GqlPartial } from "../types";

export const autoFillResult: GqlPartial<AutoFillResult> = {
    __typename: "AutoFillResult",
    common: {
        data: true,
    },
};

export const checkTaskStatusesResult: GqlPartial<CheckTaskStatusesResult> = {
    __typename: "CheckTaskStatusesResult",
    common: {
        statuses: { id: true, status: true }
    },
};