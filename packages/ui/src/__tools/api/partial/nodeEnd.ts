import { NodeEnd } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const nodeEnd: GqlPartial<NodeEnd> = {
    __typename: "NodeEnd",
    common: {
        id: true,
        wasSuccessful: true,
        suggestedNextRoutineVersions: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
    },
    full: {},
    list: {},
};
