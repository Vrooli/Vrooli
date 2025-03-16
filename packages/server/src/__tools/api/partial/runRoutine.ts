import { RunRoutine, RunRoutineYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const runRoutineYou: ApiPartial<RunRoutineYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
        canRead: true,
    },
};

export const runRoutine: ApiPartial<RunRoutine> = {
    common: {
        id: true,
        isPrivate: true,
        completedComplexity: true,
        contextSwitches: true,
        data: true,
        startedAt: true,
        timeElapsed: true,
        completedAt: true,
        name: true,
        status: true,
        ioCount: true,
        stepsCount: true,
        wasRunAutomatically: true,
        routineVersion: async () => rel((await import("./routineVersion.js")).routineVersion, "nav", { omit: "you" }),
        schedule: async () => rel((await import("./schedule.js")).schedule, "full", { omit: "runRoutine" }),
        team: async () => rel((await import("./team.js")).team, "nav"),
        user: async () => rel((await import("./user.js")).user, "nav"),
        you: () => rel(runRoutineYou, "full"),
    },
    full: {
        lastStep: true,
        io: async () => rel((await import("./runRoutineIO.js")).runRoutineIO, "list", { omit: ["runRoutine", "routineVersionInput.routineVersion", "routineVersionInput.routineVersion"] }),
        routineVersion: async () => rel((await import("./routineVersion.js")).routineVersion, "full", { omit: "you" }),
        steps: async () => rel((await import("./runRoutineStep.js")).runRoutineStep, "list"),
    },
    list: {
        lastStep: true,
    },
};
