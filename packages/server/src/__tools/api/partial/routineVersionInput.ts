import { RoutineVersionInput, RoutineVersionInputTranslation } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const routineVersionInputTranslation: ApiPartial<RoutineVersionInputTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        helpText: true,
    },
};

export const routineVersionInput: ApiPartial<RoutineVersionInput> = {
    common: {
        id: true,
        index: true,
        isRequired: true,
        name: true,
        standardVersion: async () => rel((await import("./standardVersion")).standardVersion, "list"),
        translations: () => rel(routineVersionInputTranslation, "full"),
    },
};
