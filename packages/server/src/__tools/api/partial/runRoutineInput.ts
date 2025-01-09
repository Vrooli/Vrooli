import { RunRoutineInput } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const runRoutineInput: ApiPartial<RunRoutineInput> = {
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
