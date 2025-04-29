import { RunRoutineIO } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const runRoutineIO: ApiPartial<RunRoutineIO> = {
    common: {
        id: true,
        data: true,
        routineVersionInput: {
            id: true,
            index: true,
            isRequired: true,
            name: true,
            routineVersion: async () => rel((await import("./resourceVersion.js")).routineVersion, "nav"),
            standardVersion: async () => rel((await import("./standardVersion.js")).standardVersion, "list"),
        },
        routineVersionOutput: {
            id: true,
            index: true,
            name: true,
            routineVersion: async () => rel((await import("./resourceVersion.js")).routineVersion, "nav"),
            standardVersion: async () => rel((await import("./standardVersion.js")).standardVersion, "list"),
        },
    },
};
