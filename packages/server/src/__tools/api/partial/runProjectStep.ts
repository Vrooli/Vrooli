import { RunProjectStep } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        step: true,
        directory: async () => rel((await import("./projectVersionDirectory")).projectVersionDirectory, "nav"),
    },
};
