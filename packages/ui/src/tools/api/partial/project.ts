import { Project, ProjectYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";

export const projectYou: GqlPartial<ProjectYou> = {
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

export const project: GqlPartial<Project> = {
    __typename: 'Project',
    common: {
        __define: {
            0: async () => rel((await import('./organization')).organization, 'nav'),
            1: async () => rel((await import('./user')).user, 'nav'),
            2: async () => rel((await import('./tag')).tag, 'list'),
            3: async () => rel((await import('./label')).label, 'list'),
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
        you: () => rel(projectYou, 'full'),
    },
    full: {
        parent: async () => rel((await import('./projectVersion')).projectVersion, 'nav'),
        versions: async () => rel((await import('./projectVersion')).projectVersion, 'full', { omit: 'root' }),
        stats: async () => rel((await import('./statsProject')).statsProject, 'full'),
    },
    list: {
        versions: async () => rel((await import('./projectVersion')).projectVersion, 'list', { omit: 'root' }),
    },
    nav: {
        id: true,
        isPrivate: true,
    }
}