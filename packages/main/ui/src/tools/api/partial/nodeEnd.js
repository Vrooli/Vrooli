import { rel } from "../utils";
export const nodeEnd = {
    __typename: "NodeEnd",
    common: {
        id: true,
        wasSuccessful: true,
        suggestedNextRoutineVersions: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
    },
    full: {},
    list: {},
};
//# sourceMappingURL=nodeEnd.js.map