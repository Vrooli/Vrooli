import { RunRoutine, RunRoutineYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

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
            0: [require('./organization').organizationPartial, 'nav'],
            1: [require('./user').userPartial, 'nav'],
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
        routineVersion: () => relPartial(require('./routineVersion').routineVersionPartial, 'nav', { omit: 'you' }),
        runRoutineSchedule: () => relPartial(require('./runRoutineSchedule').runRoutineSchedulePartial, 'full', { omit: 'runRoutine' }),
        user: { __use: 1 },
        you: () => relPartial(runRoutineYouPartial, 'full'),
    },
    full: {
        inputs: () => relPartial(require('./runRoutineInput').runRoutineInputPartial, 'list', { omit: ['runRoutine', 'input.routineVersion'] }),
        steps: () => relPartial(require('./runRoutineStep').runRoutineStepPartial, 'list'),
    },
    list: {},
}