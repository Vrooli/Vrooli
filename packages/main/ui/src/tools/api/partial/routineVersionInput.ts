import { RoutineVersionInput, RoutineVersionInputTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const routineVersionInputTranslation: GqlPartial<RoutineVersionInputTranslation> = {
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

export const routineVersionInput: GqlPartial<RoutineVersionInput> = {
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
