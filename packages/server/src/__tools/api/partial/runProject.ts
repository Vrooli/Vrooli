import { RunProject, RunProjectYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const runProjectYou: ApiPartial<RunProjectYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
        canRead: true,
    },
};

export const runProject: ApiPartial<RunProject> = {
    common: {
        id: true,
        isPrivate: true,
        completedComplexity: true,
        contextSwitches: true,
        startedAt: true,
        timeElapsed: true,
        completedAt: true,
        name: true,
        projectVersion: async () => rel((await import("./projectVersion.js")).projectVersion, "nav", { omit: "you" }),
        status: true,
        stepsCount: true,
        schedule: async () => rel((await import("./schedule.js")).schedule, "full", { omit: "runProject" }),
        team: async () => rel((await import("./team.js")).team, "nav"),
        user: async () => rel((await import("./user.js")).user, "nav"),
        you: () => rel(runProjectYou, "full"),
    },
    full: {
        lastStep: true,
        projectVersion: async () => rel((await import("./projectVersion.js")).projectVersion, "list", { omit: "you" }),
        steps: async () => rel((await import("./runProjectStep.js")).runProjectStep, "full", { omit: "run" }),
    },
    list: {
        lastStep: true,
    },
};
