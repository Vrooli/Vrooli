import { RunProject, RunProjectYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const runProjectYouPartial: GqlPartial<RunProjectYou> = {
    __typename: 'RunProjectYou',
    common: {
        canDelete: true,
        canEdit: true,
        canView: true,
    },
    full: {},
    list: {},
}

export const runProjectPartial: GqlPartial<RunProject> = {
    __typename: 'RunProject',
    common: {
        __define: {
            0: () => relPartial(require('./organization').organizationPartial, 'nav'),
            1: () => relPartial(require('./user').userPartial, 'nav'),
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
        wasRunAutomaticaly: true,
        organization: { __use: 0 },
        projectVersion: () => relPartial(require('./projectVersion').projectVersionPartial, 'nav', { omit: 'you' }),
        runProjectSchedule: () => relPartial(require('./runProjectSchedule').runProjectSchedulePartial, 'full', { omit: 'runProject' }),
        user: { __use: 1 },
        you: () => relPartial(runProjectYouPartial, 'full'),
    },
    full: {
        steps: () => relPartial(require('./runProjectStep').runProjectStepPartial, 'full', { omit: 'run' }),
    },
    list: {},
}