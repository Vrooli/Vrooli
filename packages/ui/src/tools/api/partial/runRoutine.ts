import { RunRoutine, RunRoutineYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const runRoutineYou: GqlPartial<RunRoutineYou> = {
    __typename: 'RunRoutineYou',
    common: {
        canDelete: true,
        canUpdate: true,
        canRead: true,
    },
    full: {},
    list: {},
}

export const runRoutine: GqlPartial<RunRoutine> = {
    __typename: 'RunRoutine',
    common: {
        __define: {
            0: async () => rel((await import('./organization')).organization, 'nav'),
            1: async () => rel((await import('./user')).user, 'nav'),
        },
        id: true,
        isPrivate: true,
        completedComplexity: true,
        contextSwitches: true,
        startedAt: true,
        timeElapsed: true,
        completedAt: true,
        name: true,
        status: true,
        stepsCount: true,
        inputsCount: true,
        wasRunAutomaticaly: true,
        organization: { __use: 0 },
        routineVersion: async () => rel((await import('./routineVersion')).routineVersion, 'nav', { omit: 'you' }),
        runRoutineSchedule: async () => rel((await import('./runRoutineSchedule')).runRoutineSchedule, 'full', { omit: 'runRoutine' }),
        user: { __use: 1 },
        you: () => rel(runRoutineYou, 'full'),
    },
    full: {
        inputs: async () => rel((await import('./runRoutineInput')).runRoutineInput, 'list', { omit: ['runRoutine', 'input.routineVersion'] }),
        steps: async () => rel((await import('./runRoutineStep')).runRoutineStep, 'list'),
    },
    list: {},
}