import { RunProject, RunProjectYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const runProjectYouPartial: GqlPartial<RunProjectYou> = {
    __typename: 'RunProjectYou',
    full: {
        canDelete: true,
        canEdit: true,
        canView: true,
    },
}

export const runProjectPartial: GqlPartial<RunProject> = {
    __typename: 'RunProject',
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
        wasRunAutomaticaly: true,
        organization: () => relPartial(require('./organization').organizationPartial, 'nav'),
        projectVersion: () => relPartial(require('./projectVersion').projectVersionPartial, 'nav', { omit: 'you' }),
        runProjectSchedule: () => relPartial(require('./runProjectSchedule').runProjectSchedulePartial, 'full', { omit: 'runProject' }),
        user: () => relPartial(require('./user').userPartial, 'nav'),
        you: () => relPartial(runProjectYouPartial, 'full'),
    },
    full: {
        steps: () => relPartial(require('./runProjectStep').runProjectStepPartial, 'full', { omit: 'run' }),
    },
}