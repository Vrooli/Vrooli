import { RunProject, RunProjectYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { organizationPartial } from "./organization";
import { projectVersionPartial } from "./projectVersion";
import { runProjectSchedulePartial } from "./runProjectSchedule";
import { runProjectStepPartial } from "./runProjectStep";
import { userPartial } from "./user";

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
        organization: () => relPartial(organizationPartial, 'nav'),
        projectVersion: () => relPartial(projectVersionPartial, 'nav', { omit: 'you' }),
        runProjectSchedule: () => relPartial(runProjectSchedulePartial, 'full', { omit: 'runProject' }),
        user: () => relPartial(userPartial, 'nav'),
        you: () => relPartial(runProjectYouPartial, 'full'),
    },
    full: {
        steps: () => relPartial(runProjectStepPartial, 'full', { omit: 'run' }),
    },
}