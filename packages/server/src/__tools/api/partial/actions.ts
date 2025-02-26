import { CopyResult } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const copyResult: ApiPartial<CopyResult> = {
    common: {
        apiVersion: async () => rel((await import("./apiVersion.js")).apiVersion, "list"),
        codeVersion: async () => rel((await import("./codeVersion.js")).codeVersion, "list"),
        noteVersion: async () => rel((await import("./noteVersion.js")).noteVersion, "list"),
        projectVersion: async () => rel((await import("./projectVersion.js")).projectVersion, "list"),
        routineVersion: async () => rel((await import("./routineVersion.js")).routineVersion, "list"),
        standardVersion: async () => rel((await import("./standardVersion.js")).standardVersion, "list"),
        team: async () => rel((await import("./team.js")).team, "list"),
    },
};
