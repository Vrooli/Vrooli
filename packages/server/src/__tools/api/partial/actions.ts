import { type CopyResult } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const copyResult: ApiPartial<CopyResult> = {
    common: {
        resourceVersion: async () => rel((await import("./resourceVersion.js")).resourceVersion, "list"),
        team: async () => rel((await import("./team.js")).team, "list"),
    },
};
