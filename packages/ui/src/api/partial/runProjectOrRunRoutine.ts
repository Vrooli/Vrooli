import { RunProjectOrRunRoutine } from "@shared/consts";
import { GqlPartial } from "types";

export const runProjectOrRunRoutinePartial: GqlPartial<RunProjectOrRunRoutine> = {
    __typename: 'RunProjectOrRunRoutine' as any,
    full: {
        __define: {
            0: [require('./runProject').runProjectPartial, 'full'],
            1: [require('./runRoutine').runRoutinePartial, 'full'],
        },
        __union: {
            RunProject: 0,
            RunRoutine: 1,
        }
    },
    list: {
        __define: {
            0: [require('./runProject').runProjectPartial, 'list'],
            1: [require('./runRoutine').runRoutinePartial, 'list'],
        },
        __union: {
            RunProject: 0,
            RunRoutine: 1,
        }
    }
}