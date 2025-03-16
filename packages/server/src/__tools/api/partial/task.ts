import { CheckTaskStatusesResult } from "@local/shared";
import { ApiPartial } from "../types.js";

export const checkTaskStatusesResult: ApiPartial<CheckTaskStatusesResult> = {
    common: {
        statuses: { id: true, status: true },
    },
};
