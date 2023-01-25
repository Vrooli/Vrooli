import { NotificationSubscription } from "@shared/consts";
import { GqlPartial } from "types";

export const notificationSubscriptionPartial: GqlPartial<NotificationSubscription> = {
    __typename: 'NotificationSubscription',
    full: {
        __define: {
            0: [require('./api').apiPartial, 'list'],
            1: [require('./comment').commentPartial, 'list'],
            2: [require('./issue').issuePartial, 'list'],
            3: [require('./meeting').meetingPartial, 'list'],
            4: [require('./note').notePartial, 'list'],
            5: [require('./organization').organizationPartial, 'list'],
            6: [require('./project').projectPartial, 'list'],
            7: [require('./pullRequest').pullRequestPartial, 'list'],
            8: [require('./question').questionPartial, 'list'],
            9: [require('./quiz').quizPartial, 'list'],
            10: [require('./report').reportPartial, 'list'],
            11: [require('./routine').routinePartial, 'list'],
            12: [require('./smartContract').smartContractPartial, 'list'],
            13: [require('./standard').standardPartial, 'list'],
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