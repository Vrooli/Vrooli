import { CopyResult } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const copyResult: ApiPartial<CopyResult> = {
    common: {
        apiVersion: async () => rel((await import("./apiVersion")).apiVersion, "list"),
        codeVersion: async () => rel((await import("./codeVersion")).codeVersion, "list"),
        noteVersion: async () => rel((await import("./noteVersion")).noteVersion, "list"),
        projectVersion: async () => rel((await import("./projectVersion")).projectVersion, "list"),
        routineVersion: async () => rel((await import("./routineVersion")).routineVersion, "list"),
        standardVersion: async () => rel((await import("./standardVersion")).standardVersion, "list"),
        team: async () => rel((await import("./team")).team, "list"),
    },
};
