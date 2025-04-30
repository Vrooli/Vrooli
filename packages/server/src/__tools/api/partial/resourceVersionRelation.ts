import { ResourceVersionRelation } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const resourceVersionRelation: ApiPartial<ResourceVersionRelation> = {
    common: {
        id: true,
        toVersion: async () => rel((await import("./resourceVersion.js")).resourceVersion, "nav", { omit: ["relatedVersions"] }),
    },
};
