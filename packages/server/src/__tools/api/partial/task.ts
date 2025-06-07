import { type CheckTaskStatusesResult } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";

export const checkTaskStatusesResult: ApiPartial<CheckTaskStatusesResult> = {
    common: {
        statuses: { id: true, status: true },
    },
};
