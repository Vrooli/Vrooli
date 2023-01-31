import { NotificationSubscription } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const notificationSubscriptionPartial: GqlPartial<NotificationSubscription> = {
    __typename: 'NotificationSubscription',
    full: {
        __define: {
            0: () => relPartial(require('./api').apiPartial, 'list'),
            1: () => relPartial(require('./comment').commentPartial, 'list'),
            2: () => relPartial(require('./issue').issuePartial, 'list'),
            3: () => relPartial(require('./meeting').meetingPartial, 'list'),
            4: () => relPartial(require('./note').notePartial, 'list'),
            5: () => relPartial(require('./organization').organizationPartial, 'list'),
            6: () => relPartial(require('./project').projectPartial, 'list'),
            7: () => relPartial(require('./pullRequest').pullRequestPartial, 'list'),
            8: () => relPartial(require('./question').questionPartial, 'list'),
            9: () => relPartial(require('./quiz').quizPartial, 'list'),
            10: () => relPartial(require('./report').reportPartial, 'list'),
            11: () => relPartial(require('./routine').routinePartial, 'list'),
            12: () => relPartial(require('./smartContract').smartContractPartial, 'list'),
            13: () => relPartial(require('./standard').standardPartial, 'list'),
        },
        id: true,
        created_at: true,
        silent: true,
        object: {
            __union: {
                Api: 0,
                Comment: 1,
                Issue: 2,
                Meeting: 3,
                Note: 4,
                Organization: 5,
                Project: 6,
                PullRequest: 7,
                Question: 8,
                Quiz: 9,
                Report: 10,
                Routine: 11,
                SmartContract: 12,
                Standard: 13,
            }
        }
    },
}