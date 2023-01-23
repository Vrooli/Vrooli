import { NodeRoutineList } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const nodeRoutineListPartial: GqlPartial<NodeRoutineList> = {
    __typename: 'NodeRoutineList',
    common: {
        id: true,
        isOrdered: true,
        isOptional: true,
        items: () => relPartial(require('./nodeRoutineListItem').nodeRoutineListItemPartial, 'full'),
    },
}