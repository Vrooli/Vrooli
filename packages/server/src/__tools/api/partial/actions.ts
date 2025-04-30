import { CopyResult } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const copyResult: ApiPartial<CopyResult> = {
    common: {
        resourceVersion: async () => rel((await import("./resourceVersion.js")).resourceVersion, "list"),
        team: async () => rel((await import("./team.js")).team, "list"),
    },
};
