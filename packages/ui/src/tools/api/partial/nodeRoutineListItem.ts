import { NodeRoutineListItem, NodeRoutineListItemTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const nodeRoutineListItemTranslationPartial: GqlPartial<NodeRoutineListItemTranslation> = {
    __typename: 'NodeRoutineListItemTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const nodeRoutineListItemPartial: GqlPartial<NodeRoutineListItem> = {
    __typename: 'NodeRoutineListItem',
    common: {
        id: true,
        index: true,
        isOptional: true,
        translations: () => relPartial(nodeRoutineListItemTranslationPartial, 'full'),
    },
    full: {},
    list: {},
}