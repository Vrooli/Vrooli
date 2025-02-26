import { RunProjectStep } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const runProjectStep: ApiPartial<RunProjectStep> = {
    common: {
        id: true,
        order: true,
        contextSwitches: true,
        startedAt: true,
        timeElapsed: true,
        completedAt: true,
        name: true,
        status: true,
        directory: async () => rel((await import("./projectVersionDirectory.js")).projectVersionDirectory, "nav"),
    },
};
