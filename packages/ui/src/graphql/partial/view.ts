import { View } from "@shared/consts";
import { GqlPartial } from "types";
import { apiPartial } from "./api";
import { issuePartial } from "./issue";
import { notePartial } from "./note";
import { organizationPartial } from "./organization";
import { postPartial } from "./post";
import { projectPartial } from "./project";
import { questionPartial } from "./question";
import { routinePartial } from "./routine";
import { smartContractPartial } from "./smartContract";
import { standardPartial } from "./standard";
import { userPartial } from "./user";

export const viewPartial: GqlPartial<View> = {
    __typename: 'View',
    list: () => ({
        __define: {
            0: [apiPartial, 'list'],
            1: [issuePartial, 'list'],
            2: [notePartial, 'list'],
            3: [organizationPartial, 'list'],
            4: [postPartial, 'list'],
            5: [projectPartial, 'list'],
            6: [questionPartial, 'list'],
            7: [routinePartial, 'list'],
            8: [smartContractPartial, 'list'],
            9: [standardPartial, 'list'],
            10: [userPartial, 'list']
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
    })
}