import { RunRoutineStep } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const runRoutineStepPartial: GqlPartial<RunRoutineStep> = {
    __typename: 'RunRoutineStep',
    common: {
        id: true,
        order: true,
        contextSwitches: true,
        startedAt: true,
        timeElapsed: true,
        completedAt: true,
        name: true,
        status: true,
        step: true,
        subroutine: async () => relPartial((await import('./routineVersion')).routineVersionPartial, 'nav'),
    },
    full: {},
    list: {},
}