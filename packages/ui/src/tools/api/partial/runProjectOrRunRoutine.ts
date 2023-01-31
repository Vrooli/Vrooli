import { RunProjectOrRunRoutine } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const runProjectOrRunRoutinePartial: GqlPartial<RunProjectOrRunRoutine> = {
    __typename: 'RunProjectOrRunRoutine' as any,
    full: {
        __define: {
            0: async () => relPartial((await import('./runProject')).runProjectPartial, 'full'),
            1: async () => relPartial((await import('./runRoutine')).runRoutinePartial, 'full'),
        },
        __union: {
            RunProject: 0,
            RunRoutine: 1,
        }
    },
    list: {
        __define: {
            0: async () => relPartial((await import('./runProject')).runProjectPartial, 'list'),
            1: async () => relPartial((await import('./runRoutine')).runRoutinePartial, 'list'),
        },
        __union: {
            RunProject: 0,
            RunRoutine: 1,
        }
    }
}