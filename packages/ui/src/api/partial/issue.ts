import { Issue, IssueTranslation, IssueYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const issueTranslationPartial: GqlPartial<IssueTranslation> = {
    __typename: 'IssueTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
}

export const issueYouPartial: GqlPartial<IssueYou> = {
    __typename: 'IssueYou',
    full: {
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
}

export const issuePartial: GqlPartial<Issue> = {
    __typename: 'Issue',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        closedAt: true,
        referencedVersionId: true,
        status: true,
        to: {
            __union: {
                Api: () => relPartial(require('./api').apiPartial, 'nav'),
                Note: () => relPartial(require('./note').notePartial, 'nav'),
                Organization: () => relPartial(require('./organization').organizationPartial, 'nav'),
                Project: () => relPartial(require('./project').projectPartial, 'nav'),
                Routine: () => relPartial(require('./routine').routinePartial, 'nav'),
                SmartContract: () => relPartial(require('./smartContract').smartContractPartial, 'nav'),
                Standard: () => relPartial(require('./standard').standardPartial, 'nav'),
            }
        },
        commentsCount: true,
        reportsCount: true,
        score: true,
        stars: true,
        views: true,
        labels: () => relPartial(require('./label').labelPartial, 'list'),
        you: () => relPartial(issueYouPartial, 'full'),
    },
    full: {
        closedBy: () => relPartial(require('./user').userPartial, 'nav'),
        createdBy: () => relPartial(require('./user').userPartial, 'nav'),
        translations: () => relPartial(issueTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(issueTranslationPartial, 'list'),
    }
}