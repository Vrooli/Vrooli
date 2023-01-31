import { Issue, IssueTranslation, IssueYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const issueTranslationPartial: GqlPartial<IssueTranslation> = {
    __typename: 'IssueTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const issueYouPartial: GqlPartial<IssueYou> = {
    __typename: 'IssueYou',
    common: {
        canComment: true,
        canDelete: true,
        canEdit: true,
        canStar: true,
        canReport: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
    },
    full: {},
    list: {},
}

export const issuePartial: GqlPartial<Issue> = {
    __typename: 'Issue',
    common: {
        __define: {
            0: async () => relPartial((await import('./api')).api, 'nav'),
            1: async () => relPartial((await import('./note')).notePartial, 'nav'),
            2: async () => relPartial((await import('./organization')).organizationPartial, 'nav'),
            3: async () => relPartial((await import('./project')).projectPartial, 'nav'),
            4: async () => relPartial((await import('./routine')).routinePartial, 'nav'),
            5: async () => relPartial((await import('./smartContract')).smartContractPartial, 'nav'),
            6: async () => relPartial((await import('./standard')).standardPartial, 'nav'),
            7: async () => relPartial((await import('./label')).labelPartial, 'nav'),
        },
        id: true,
        created_at: true,
        updated_at: true,
        closedAt: true,
        referencedVersionId: true,
        status: true,
        to: {
            __union: {
                Api: 0,
                Note: 1,
                Organization: 2,
                Project: 3,
                Routine: 4,
                SmartContract: 5,
                Standard: 6,
            }
        },
        commentsCount: true,
        reportsCount: true,
        score: true,
        stars: true,
        views: true,
        labels: { __use: 7 },
        you: () => relPartial(issueYouPartial, 'full'),
    },
    full: {
        closedBy: async () => relPartial((await import('./user')).userPartial, 'nav'),
        createdBy: async () => relPartial((await import('./user')).userPartial, 'nav'),
        translations: () => relPartial(issueTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(issueTranslationPartial, 'list'),
    },
    nav: {
        id: true,
        translations: () => relPartial(issueTranslationPartial, 'list'),
    }
}