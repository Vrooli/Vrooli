import { RunProject, RunProjectYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const runProjectYou: GqlPartial<RunProjectYou> = {
    __typename: "RunProjectYou",
    common: {
        canDelete: true,
        canUpdate: true,
        canRead: true,
    },
    full: {},
    list: {},
};

export const runProject: GqlPartial<RunProject> = {
    __typename: "RunProject",
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
        schedule: async () => rel((await import("./schedule")).schedule, "full", { omit: "runProject" }),
        team: { __use: 0 },
        user: { __use: 1 },
        you: () => rel(runProjectYou, "full"),
    },
    full: {
        projectVersion: async () => rel((await import("./projectVersion")).projectVersion, "list", { omit: "you" }),
        steps: async () => rel((await import("./runProjectStep")).runProjectStep, "full", { omit: "run" }),
    },
    list: {
        projectVersion: async () => rel((await import("./projectVersion")).projectVersion, "nav", { omit: "you" }),
    },
};
