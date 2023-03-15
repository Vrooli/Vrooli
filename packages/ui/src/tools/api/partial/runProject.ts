import { RunProject, RunProjectYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const runProjectYou: GqlPartial<RunProjectYou> = {
    __typename: 'RunProjectYou',
    common: {
        canDelete: true,
        canUpdate: true,
        canRead: true,
    },
    full: {},
    list: {},
}

export const runProject: GqlPartial<RunProject> = {
    __typename: 'RunProject',
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
        organization: { __use: 0 },
        projectVersion: async () => rel((await import('./projectVersion')).projectVersion, 'nav', { omit: 'you' }),
        runProjectSchedule: async () => rel((await import('./runProjectSchedule')).runProjectSchedule, 'full', { omit: 'runProject' }),
        user: { __use: 1 },
        you: () => rel(runProjectYou, 'full'),
    },
    full: {
        steps: async () => rel((await import('./runProjectStep')).runProjectStep, 'full', { omit: 'run' }),
    },
    list: {},
}