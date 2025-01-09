import { NodeEnd } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const nodeEnd: ApiPartial<NodeEnd> = {
    common: {
        id: true,
        wasSuccessful: true,
        suggestedNextRoutineVersions: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
    },
};
