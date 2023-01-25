import { RunRoutine, RunRoutineYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const runRoutineYouPartial: GqlPartial<RunRoutineYou> = {
    __typename: 'RunRoutineYou',
    full: {
        canDelete: true,
        canEdit: true,
        canView: true,
    },
}

export const runRoutinePartial: GqlPartial<RunRoutine> = {
    __typename: 'RunRoutine',
    common: {
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
        organization: () => relPartial(require('./organization').organizationPartial, 'nav'),
        routineVersion: () => relPartial(require('./routineVersion').routineVersionPartial, 'nav', { omit: 'you' }),
        runRoutineSchedule: () => relPartial(require('./runRoutineSchedule').runRoutineSchedulePartial, 'full', { omit: 'runRoutine' }),
        user: () => relPartial(require('./user').userPartial, 'nav'),
        you: () => relPartial(runRoutineYouPartial, 'full'),
    },
    full: {
        inputs: () => relPartial(require('./runRoutineInput').runRoutineInputPartial, 'list', { omit: 'runRoutine' }),
        steps: () => relPartial(require('./runRoutineStep').runRoutineStepPartial, 'list'),
    },
}