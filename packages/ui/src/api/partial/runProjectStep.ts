import { RunProjectStep } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const runProjectStepPartial: GqlPartial<RunProjectStep> = {
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
        directory: () => relPartial(require('./projectVersionDirectory').projectVersionDirectoryPartial, 'nav')
    },
}