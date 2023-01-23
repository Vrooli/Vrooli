import { PopularResult } from "@shared/consts";
import { GqlPartial } from "types";

export const popularResultPartial: GqlPartial<PopularResult> = {
    __typename: 'PopularResult',
    list: {
        __define: {
            0: [require('./organization').organizationPartial, 'list'],
            1: [require('./project').projectPartial, 'list'],
            2: [require('./routine').routinePartial, 'list'],
            3: [require('./standard').standardPartial, 'list'],
            4: [require('./user').userPartial, 'list'],
        },
        organizations: { __use: 0 },
        projects: { __use: 1 },
        routines: { __use: 2 },
        standards: { __use: 3 },
        users: { __use: 4 },
    }
}