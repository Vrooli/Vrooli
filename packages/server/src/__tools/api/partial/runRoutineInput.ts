import { RunRoutineInput } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const runRoutineInput: GqlPartial<RunRoutineInput> = {
    common: {
        id: true,
        data: true,
        input: {
            id: true,
            index: true,
            isRequired: true,
            name: true,
            routineVersion: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
            standardVersion: async () => rel((await import("./standardVersion")).standardVersion, "list"),
        },
    },
};
