import { RunRoutineOutput } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const runRoutineOutput: GqlPartial<RunRoutineOutput> = {
    common: {
        id: true,
        data: true,
        output: {
            id: true,
            index: true,
            name: true,
            routineVersion: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
            standardVersion: async () => rel((await import("./standardVersion")).standardVersion, "list"),
        },
    },
};
