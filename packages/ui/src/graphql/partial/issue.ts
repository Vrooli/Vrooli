import { Issue, IssueTranslation, IssueYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { apiPartial } from "./api";
import { labelPartial } from "./label";
import { notePartial } from "./note";
import { organizationPartial } from "./organization";
import { projectPartial } from "./project";
import { routinePartial } from "./routine";
import { smartContractPartial } from "./smartContract";
import { standardPartial } from "./standard";
import { userPartial } from "./user";

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
                Api: () => relPartial(apiPartial, 'nav'),
                Note: () => relPartial(notePartial, 'nav'),
                Organization: () => relPartial(organizationPartial, 'nav'),
                Project: () => relPartial(projectPartial, 'nav'),
                Routine: () => relPartial(routinePartial, 'nav'),
                SmartContract: () => relPartial(smartContractPartial, 'nav'),
                Standard: () => relPartial(standardPartial, 'nav'),
            }
        },
        commentsCount: true,
        reportsCount: true,
        score: true,
        stars: true,
        views: true,
        labels: () => relPartial(labelPartial, 'list'),
        you: () => relPartial(issueYouPartial, 'full'),
    },
    full: {
        closedBy: () => relPartial(userPartial, 'nav'),
        createdBy: () => relPartial(userPartial, 'nav'),
        translations: () => relPartial(issueTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(issueTranslationPartial, 'list'),
    }
}