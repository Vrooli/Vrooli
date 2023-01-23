import { Project, ProjectYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const projectYouPartial: GqlPartial<ProjectYou> = {
    __typename: 'ProjectYou',
    full: {
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
}

export const projectPartial: GqlPartial<Project> = {
    __typename: 'Project',
    common: {
        id: true,
        created_at: true,
        isPrivate: true,
        issuesCount: true,
        labels: () => relPartial(require('./label').labelPartial, 'list'),
        owner: {
            __union: {
                Organization: () => relPartial(require('./organization').organizationPartial, 'nav'),
                User: () => relPartial(require('./user').userPartial, 'nav'),
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: () => relPartial(require('./tag').tagPartial, 'list'),
        transfersCount: true,
        views: true,
        you: () => relPartial(projectYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(require('./projectVersion').projectVersionPartial, 'full', { omit: 'root' }),
        stats: () => relPartial(require('./statsProject').statsProjectPartial, 'full'),
    },
    list: {
        versions: () => relPartial(require('./projectVersion').projectVersionPartial, 'list'),
    }
}