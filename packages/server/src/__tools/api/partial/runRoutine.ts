import { RunRoutine, RunRoutineYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const runRoutineYou: GqlPartial<RunRoutineYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
        canRead: true,
    },
};

export const runRoutine: GqlPartial<RunRoutine> = {
    common: {
        id: true,
        isPrivate: true,
        completedComplexity: true,
        contextSwitches: true,
        startedAt: true,
        timeElapsed: true,
        completedAt: true,
        name: true,
        status: true,
        inputsCount: true,
        outputsCount: true,
        stepsCount: true,
        wasRunAutomatically: true,
        routineVersion: async () => rel((await import("./routineVersion")).routineVersion, "nav", { omit: "you" }),
        schedule: async () => rel((await import("./schedule")).schedule, "full", { omit: "runRoutine" }),
        team: async () => rel((await import("./team")).team, "nav"),
        user: async () => rel((await import("./user")).user, "nav"),
        you: () => rel(runRoutineYou, "full"),
    },
    full: {
        lastStep: true,
        inputs: async () => rel((await import("./runRoutineInput")).runRoutineInput, "list", { omit: ["runRoutine", "input.routineVersion"] }),
        outputs: async () => rel((await import("./runRoutineOutput")).runRoutineOutput, "list", { omit: ["runRoutine", "output.routineVersion"] }),
        routineVersion: async () => rel((await import("./routineVersion")).routineVersion, "full", { omit: "you" }),
        steps: async () => rel((await import("./runRoutineStep")).runRoutineStep, "list"),
    },
    list: {
        lastStep: true,
    },
};
