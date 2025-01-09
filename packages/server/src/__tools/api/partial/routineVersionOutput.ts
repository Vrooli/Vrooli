import { RoutineVersionOutput, RoutineVersionOutputTranslation } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        standardVersion: async () => rel((await import("./standardVersion")).standardVersion, "list"),
        translations: () => rel(routineVersionOutputTranslation, "full"),
    },
};
