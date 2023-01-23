import { NodeRoutineListItem, NodeRoutineListItemTranslation } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const nodeRoutineListItemTranslationPartial: GqlPartial<NodeRoutineListItemTranslation> = {
    __typename: 'NodeRoutineListItemTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
}

export const nodeRoutineListItemPartial: GqlPartial<NodeRoutineListItem> = {
    __typename: 'NodeRoutineListItem',
    common: {
        id: true,
        index: true,
        isOptional: true,
        translations: () => relPartial(nodeRoutineListItemTranslationPartial, 'full'),
    },
}