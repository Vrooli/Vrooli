import { NodeRoutineList } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const nodeRoutineList: GqlPartial<NodeRoutineList> = {
    __typename: 'NodeRoutineList',
    common: {
        id: true,
        isOrdered: true,
        isOptional: true,
        items: async () => rel((await import('./nodeRoutineListItem')).nodeRoutineListItem, 'full'),
    },
    full: {},
    list: {},
}