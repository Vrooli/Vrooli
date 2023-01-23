import { DevelopResult } from "@shared/consts";
import { GqlPartial } from "types";

export const developResultPartial: GqlPartial<DevelopResult> = {
    __typename: 'DevelopResult',
    list: {
        __define: {
            0: [require('./project').projectPartial, 'list'],
            1: [require('./routine').routinePartial, 'list'],
        },
        completed: {
            __union: {
                Project: 0,
                Routine: 1,
            }
        },
        inProgress: {
            __union: {
                Project: 0,
                Routine: 1,
            }
        },
        recent: {
            __union: {
                Project: 0,
                Routine: 1,
            }
        },
    }
}