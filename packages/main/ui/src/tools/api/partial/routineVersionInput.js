import { rel } from "../utils";
export const routineVersionInputTranslation = {
    __typename: "RoutineVersionInputTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        helpText: true,
    },
    full: {},
    list: {},
};
export const routineVersionInput = {
    __typename: "RoutineVersionInput",
    common: {
        id: true,
        index: true,
        isRequired: true,
        name: true,
        standardVersion: async () => rel((await import("./standardVersion")).standardVersion, "list"),
        translations: () => rel(routineVersionInputTranslation, "full"),
    },
    full: {},
    list: {},
};
//# sourceMappingURL=routineVersionInput.js.map