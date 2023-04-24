import { NodeRoutineListItem, NodeRoutineListItemTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const nodeRoutineListItemTranslation: GqlPartial<NodeRoutineListItemTranslation> = {
    __typename: "NodeRoutineListItemTranslation",
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
};

export const nodeRoutineListItem: GqlPartial<NodeRoutineListItem> = {
    __typename: "NodeRoutineListItem",
    common: {
        id: true,
        index: true,
        isOptional: true,
        translations: () => rel(nodeRoutineListItemTranslation, "full"),
    },
    full: {},
    list: {},
};
