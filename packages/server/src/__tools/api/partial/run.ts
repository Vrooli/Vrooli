import { type Run, type RunYou } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const runYou: ApiPartial<RunYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
        canRead: true,
    },
};

export const run: ApiPartial<Run> = {
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
        resourceVersion: async () => rel((await import("./resourceVersion.js")).resourceVersion, "nav", { omit: "you" }),
        schedule: async () => rel((await import("./schedule.js")).schedule, "full", { omit: "run" }),
        team: async () => rel((await import("./team.js")).team, "nav"),
        user: async () => rel((await import("./user.js")).user, "nav"),
        you: () => rel(runYou, "full"),
    },
    full: {
        lastStep: true,
        io: async () => rel((await import("./runIO.js")).runIO, "list", { omit: ["run"] }),
        resourceVersion: async () => rel((await import("./resourceVersion.js")).resourceVersion, "full", { omit: "you" }),
        steps: async () => rel((await import("./runStep.js")).runStep, "list"),
    },
    list: {
        lastStep: true,
    },
};
