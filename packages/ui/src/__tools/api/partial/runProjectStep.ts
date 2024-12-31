import { RunProjectStep } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const runProjectStep: GqlPartial<RunProjectStep> = {
    __typename: "RunProjectStep",
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
    full: {},
    list: {},
};
