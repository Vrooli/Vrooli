import { RunProject, RunProjectYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        projectVersion: async () => rel((await import("./projectVersion")).projectVersion, "nav", { omit: "you" }),
        status: true,
        stepsCount: true,
        schedule: async () => rel((await import("./schedule")).schedule, "full", { omit: "runProject" }),
        team: async () => rel((await import("./team")).team, "nav"),
        user: async () => rel((await import("./user")).user, "nav"),
        you: () => rel(runProjectYou, "full"),
    },
    full: {
        lastStep: true,
        projectVersion: async () => rel((await import("./projectVersion")).projectVersion, "list", { omit: "you" }),
        steps: async () => rel((await import("./runProjectStep")).runProjectStep, "full", { omit: "run" }),
    },
    list: {
        lastStep: true,
    },
};
