import { NodeRoutineListItem, NodeRoutineListItemTranslation } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

export const nodeRoutineListItemTranslation: ApiPartial<NodeRoutineListItemTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const nodeRoutineListItem: ApiPartial<NodeRoutineListItem> = {
    common: {
        id: true,
        index: true,
        isOptional: true,
        translations: () => rel(nodeRoutineListItemTranslation, "full"),
    },
    full: {
        routineVersion: async () => rel((await import("./routineVersion")).routineVersion, "nav"),
    },
};
