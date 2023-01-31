import { RunRoutine, RunRoutineYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const runRoutineYouPartial: GqlPartial<RunRoutineYou> = {
    __typename: 'RunRoutineYou',
    common: {
        canDelete: true,
        canEdit: true,
        canView: true,
    },
    full: {},
    list: {},
}

export const runRoutinePartial: GqlPartial<RunRoutine> = {
    __typename: 'RunRoutine',
    common: {
        __define: {
            0: async () => relPartial((await import('./organization')).organizationPartial, 'nav'),
            1: async () => relPartial((await import('./user')).userPartial, 'nav'),
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
        routineVersion: async () => relPartial((await import('./routineVersion')).routineVersionPartial, 'nav', { omit: 'you' }),
        runRoutineSchedule: async () => relPartial((await import('./runRoutineSchedule')).runRoutineSchedulePartial, 'full', { omit: 'runRoutine' }),
        user: { __use: 1 },
        you: () => relPartial(runRoutineYouPartial, 'full'),
    },
    full: {
        inputs: async () => relPartial((await import('./runRoutineInput')).runRoutineInputPartial, 'list', { omit: ['runRoutine', 'input.routineVersion'] }),
        steps: async () => relPartial((await import('./runRoutineStep')).runRoutineStepPartial, 'list'),
    },
    list: {},
}