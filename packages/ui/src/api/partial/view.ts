import { View } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "types";

export const viewPartial: GqlPartial<View> = {
    __typename: 'View',
    list: {
        __define: {
            0: () => relPartial(require('./api').apiPartial, 'list'),
            1: () => relPartial(require('./issue').issuePartial, 'list'),
            2: () => relPartial(require('./note').notePartial, 'list'),
            3: () => relPartial(require('./organization').organizationPartial, 'list'),
            4: () => relPartial(require('./post').postPartial, 'list'),
            5: () => relPartial(require('./project').projectPartial, 'list'),
            6: () => relPartial(require('./question').questionPartial, 'list'),
            7: () => relPartial(require('./routine').routinePartial, 'list'),
            8: () => relPartial(require('./smartContract').smartContractPartial, 'list'),
            9: () => relPartial(require('./standard').standardPartial, 'list'),
            10: () => relPartial(require('./user').userPartial, 'list'),
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