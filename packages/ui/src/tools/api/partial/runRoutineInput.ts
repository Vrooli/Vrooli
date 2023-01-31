import { RunRoutineInput } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

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
            routineVersion: async () => relPartial((await import('./routineVersion')).routineVersionPartial, 'nav'),
            standardVersion: async () => relPartial((await import('./standardVersion')).standardVersionPartial, 'list'),
        },
    },
    full: {},
    list: {},
}