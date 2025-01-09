import { RunRoutineStep } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        step: true,
        subroutine: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
    },
};
