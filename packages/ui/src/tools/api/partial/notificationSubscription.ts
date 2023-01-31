import { NotificationSubscription } from "@shared/consts";
import { relPartial } from "../utils";
import { GqlPartial } from "../types";

export const notificationSubscriptionPartial: GqlPartial<NotificationSubscription> = {
    __typename: 'NotificationSubscription',
    full: {
        __define: {
            0: async () => relPartial((await import('./api')).api, 'list'),
            1: async () => relPartial((await import('./comment')).comment, 'list'),
            2: async () => relPartial((await import('./issue')).issuePartial, 'list'),
            3: async () => relPartial((await import('./meeting')).meetingPartial, 'list'),
            4: async () => relPartial((await import('./note')).notePartial, 'list'),
            5: async () => relPartial((await import('./organization')).organizationPartial, 'list'),
            6: async () => relPartial((await import('./project')).projectPartial, 'list'),
            7: async () => relPartial((await import('./pullRequest')).pullRequestPartial, 'list'),
            8: async () => relPartial((await import('./question')).questionPartial, 'list'),
            9: async () => relPartial((await import('./quiz')).quizPartial, 'list'),
            10: async () => relPartial((await import('./report')).reportPartial, 'list'),
            11: async () => relPartial((await import('./routine')).routinePartial, 'list'),
            12: async () => relPartial((await import('./smartContract')).smartContractPartial, 'list'),
            13: async () => relPartial((await import('./standard')).standardPartial, 'list'),
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