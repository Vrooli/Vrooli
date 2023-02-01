import { NodeRoutineListItem, NodeRoutineListItemTranslation } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const nodeRoutineListItemTranslation: GqlPartial<NodeRoutineListItemTranslation> = {
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

export const nodeRoutineListItem: GqlPartial<NodeRoutineListItem> = {
    __typename: 'NodeRoutineListItem',
    common: {
        id: true,
        index: true,
        isOptional: true,
        translations: () => rel(nodeRoutineListItemTranslation, 'full'),
    },
    full: {},
    list: {},
}