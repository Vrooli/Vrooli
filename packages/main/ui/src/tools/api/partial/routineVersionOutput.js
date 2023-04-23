import { rel } from "../utils";
export const routineVersionOutputTranslation = {
    __typename: "RoutineVersionOutputTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        helpText: true,
    },
    full: {},
    list: {},
};
export const routineVersionOutput = {
    __typename: "RoutineVersionOutput",
    common: {
        id: true,
        index: true,
        name: true,
        standardVersion: async () => rel((await import("./standardVersion")).standardVersion, "list"),
        translations: () => rel(routineVersionOutputTranslation, "full"),
    },
    full: {},
    list: {},
};
//# sourceMappingURL=routineVersionOutput.js.map