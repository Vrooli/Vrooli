import { RunRoutineInput } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const runRoutineInput: GqlPartial<RunRoutineInput> = {
    __typename: 'RunRoutineInput',
    common: {
        id: true,
        data: true,
        input: {
            id: true,
            index: true,
            isRequired: true,
            name: true,
            routineVersion: async () => rel((await import('./routineVersion')).routineVersion, 'nav'),
            standardVersion: async () => rel((await import('./standardVersion')).standardVersion, 'list'),
        },
    },
    full: {},
    list: {},
}