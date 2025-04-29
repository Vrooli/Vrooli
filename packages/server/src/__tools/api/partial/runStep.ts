import { RunRoutineStep } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const runRoutineStep: ApiPartial<RunRoutineStep> = {
    common: {
        id: true,
        order: true,
        contextSwitches: true,
        startedAt: true,
        timeElapsed: true,
        completedAt: true,
        name: true,
        status: true,
        subroutine: async () => rel((await import("./resourceVersion.js")).routineVersion, "nav"),
    },
};
