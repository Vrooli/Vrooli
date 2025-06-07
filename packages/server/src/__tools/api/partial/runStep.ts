import { type RunStep } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const runStep: ApiPartial<RunStep> = {
    common: {
        id: true,
        order: true,
        contextSwitches: true,
        startedAt: true,
        timeElapsed: true,
        completedAt: true,
        name: true,
        status: true,
        resourceInId: true,
        resourceVersion: async () => rel((await import("./resourceVersion.js")).resourceVersion, "nav"),
    },
};
