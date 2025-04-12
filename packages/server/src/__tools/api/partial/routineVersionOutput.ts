import { RoutineVersionOutput, RoutineVersionOutputTranslation } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const routineVersionOutputTranslation: ApiPartial<RoutineVersionOutputTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        helpText: true,
    },
};

export const routineVersionOutput: ApiPartial<RoutineVersionOutput> = {
    common: {
        id: true,
        index: true,
        name: true,
        standardVersion: async () => rel((await import("./standardVersion.js")).standardVersion, "list"),
        translations: () => rel(routineVersionOutputTranslation, "full"),
    },
};
