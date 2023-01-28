import { Project, ProjectYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const projectYouPartial: GqlPartial<ProjectYou> = {
    __typename: 'ProjectYou',
    common: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canTransfer: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    },
    full: {},
    list: {},
}

export const projectPartial: GqlPartial<Project> = {
    __typename: 'Project',
    common: {
        __define: {
            0: [require('./organization').organizationPartial, 'nav'],
            1: [require('./user').userPartial, 'nav'],
            2: [require('./tag').tagPartial, 'list'],
            3: [require('./label').labelPartial, 'list'],
        },
        id: true,
        created_at: true,
        updated_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: { __use: 3 },
        owner: {
            __union: {
                Organization: 0,
                User: 1,
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: { __use: 2 },
        transfersCount: true,
        views: true,
        you: () => relPartial(projectYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(require('./projectVersion').projectVersionPartial, 'full', { omit: 'root' }),
        stats: () => relPartial(require('./statsProject').statsProjectPartial, 'full'),
    },
    list: {
        versions: () => relPartial(require('./projectVersion').projectVersionPartial, 'list', { omit: 'root' }),
    },
    nav: {
        id: true,
        isPrivate: true,
    }
}