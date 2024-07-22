import { RunRoutine, RunRoutineYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const runRoutineYou: GqlPartial<RunRoutineYou> = {
    __typename: "RunRoutineYou",
    common: {
        canDelete: true,
        canUpdate: true,
        canRead: true,
    },
    full: {},
    list: {},
};

export const runRoutine: GqlPartial<RunRoutine> = {
    __typename: "RunRoutine",
    common: {
        __define: {
            0: async () => rel((await import("./team")).team, "nav"),
            1: async () => rel((await import("./user")).user, "nav"),
        },
        id: true,
        isPrivate: true,
        completedComplexity: true,
        contextSwitches: true,
        startedAt: true,
        timeElapsed: true,
        completedAt: true,
        name: true,
        status: true,
        stepsCount: true,
        inputsCount: true,
        wasRunAutomatically: true,
        schedule: async () => rel((await import("./schedule")).schedule, "full", { omit: "runRoutine" }),
        team: { __use: 0 },
        user: { __use: 1 },
        you: () => rel(runRoutineYou, "full"),
    },
    full: {
        lastStep: true,
        inputs: async () => rel((await import("./runRoutineInput")).runRoutineInput, "list", { omit: ["runRoutine", "input.routineVersion"] }),
        routineVersion: async () => rel((await import("./routineVersion")).routineVersion, "list", { omit: "you" }),
        steps: async () => rel((await import("./runRoutineStep")).runRoutineStep, "list"),
    },
    list: {
        lastStep: true,
        routineVersion: async () => rel((await import("./routineVersion")).routineVersion, "nav", { omit: "you" }),
    },
};
