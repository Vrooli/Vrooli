import { View } from "@shared/consts";
import { GqlPartial } from "types";

export const viewPartial: GqlPartial<View> = {
    __typename: 'View',
    list: {
        __define: {
            0: [require('./api').apiPartial, 'list'],
            1: [require('./issue').issuePartial, 'list'],
            2: [require('./note').notePartial, 'list'],
            3: [require('./organization').organizationPartial, 'list'],
            4: [require('./post').postPartial, 'list'],
            5: [require('./project').projectPartial, 'list'],
            6: [require('./question').questionPartial, 'list'],
            7: [require('./routine').routinePartial, 'list'],
            8: [require('./smartContract').smartContractPartial, 'list'],
            9: [require('./standard').standardPartial, 'list'],
            10: [require('./user').userPartial, 'list']
        },
        id: true,
        to: {
            __union: {
                Api: 0,
                Issue: 1,
                Note: 2,
                Organization: 3,
                Post: 4,
                Project: 5,
                Question: 6,
                Routine: 7,
                SmartContract: 8,
                Standard: 9,
                User: 10
            }
        }
    }
}