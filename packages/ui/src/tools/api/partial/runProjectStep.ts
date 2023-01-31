import { RunProjectStep } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const runProjectStep: GqlPartial<RunProjectStep> = {
    __typename: 'RunProjectStep',
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
        directory: async () => rel((await import('./projectVersionDirectory')).projectVersionDirectory, 'nav')
    },
    full: {},
    list: {},
}