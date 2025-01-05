import { CheckTaskStatusesResult } from "@local/shared";
import { GqlPartial } from "../types";

export const checkTaskStatusesResult: GqlPartial<CheckTaskStatusesResult> = {
    common: {
        statuses: { id: true, status: true },
    },
};
