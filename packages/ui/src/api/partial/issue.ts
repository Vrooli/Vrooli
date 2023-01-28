import { Issue, IssueTranslation, IssueYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

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
            0: [require('./api').apiPartial, 'nav'],
            1: [require('./note').notePartial, 'nav'],
            2: [require('./organization').organizationPartial, 'nav'],
            3: [require('./project').projectPartial, 'nav'],
            4: [require('./routine').routinePartial, 'nav'],
            5: [require('./smartContract').smartContractPartial, 'nav'],
            6: [require('./standard').standardPartial, 'nav'],
            7: [require('./label').labelPartial, 'list'],
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
        closedBy: () => relPartial(require('./user').userPartial, 'nav'),
        createdBy: () => relPartial(require('./user').userPartial, 'nav'),
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