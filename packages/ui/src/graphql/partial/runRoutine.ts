import { RunRoutine, RunRoutineYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { organizationPartial } from "./organization";
import { routineVersionPartial } from "./routineVersion";
import { runRoutineInputPartial } from "./runRoutineInput";
import { runRoutineSchedulePartial } from "./runRoutineSchedule";
import { runRoutineStepPartial } from "./runRoutineStep";
import { userPartial } from "./user";

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
        organization: () => relPartial(organizationPartial, 'nav'),
        routineVersion: () => relPartial(routineVersionPartial, 'nav', { omit: 'you' }),
        runRoutineSchedule: () => relPartial(runRoutineSchedulePartial, 'full', { omit: 'runRoutine' }),
        user: () => relPartial(userPartial, 'nav'),
        you: () => relPartial(runRoutineYouPartial, 'full'),
    },
    full: {
        inputs: () => relPartial(runRoutineInputPartial, 'list', { omit: 'runRoutine' }),
        steps: () => relPartial(runRoutineStepPartial, 'list'),
    },
}