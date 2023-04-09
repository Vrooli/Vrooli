import { RunProjectOrRunRoutine } from "@shared/consts";
import { rel } from "../utils";
import { GqlPartial } from "../types";

export const runProjectOrRunRoutine: GqlPartial<RunProjectOrRunRoutine> = {
    __typename: 'RunProjectOrRunRoutine' as any,
    full: {
        __define: {
            0: async () => rel((await import('./runProject')).runProject, 'full'),
            1: async () => rel((await import('./runRoutine')).runRoutine, 'full'),
        },
        __union: {
            RunProject: 0,
            RunRoutine: 1,
        }
    },
    list: {
        __define: {
            0: async () => rel((await import('./runProject')).runProject, 'list'),
            1: async () => rel((await import('./runRoutine')).runRoutine, 'list'),
        },
        __union: {
            RunProject: 0,
            RunRoutine: 1,
        }
    }
}