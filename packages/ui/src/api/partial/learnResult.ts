import { LearnResult } from "@shared/consts";
import { GqlPartial } from "types";

export const learnResultPartial: GqlPartial<LearnResult> = {
    __typename: 'LearnResult',
    list: {
        __define: {
            0: [require('./project').projectPartial, 'list'],
            1: [require('./routine').routinePartial, 'list'],
        },
        courses: { __use: 0 },
        tutorials: { __use: 1 },
    }
}