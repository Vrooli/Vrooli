import { RunRoutineInput } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const runRoutineInputPartial: GqlPartial<RunRoutineInput> = {
    __typename: 'RunRoutineInput',
    common: {
        id: true,
        data: true,
        input: {
            id: true,
            index: true,
            isRequired: true,
            name: true,
            routineVersion: () => relPartial(require('./routineVersion').routineVersionPartial, 'nav'),
            standardVersion: () => relPartial(require('./standardVersion').standardVersionPartial, 'list'),
        },
    },
    full: {},
    list: {},
}