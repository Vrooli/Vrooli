import { Project, ProjectYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { labelPartial } from "./label";
import { organizationPartial } from "./organization";
import { projectVersionPartial } from "./projectVersion";
import { statsProjectPartial } from "./statsProject";
import { tagPartial } from "./tag";
import { userPartial } from "./user";

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
        labels: () => relPartial(labelPartial, 'list'),
        owner: {
            __union: {
                Organization: () => relPartial(organizationPartial, 'nav'),
                User: () => relPartial(userPartial, 'nav'),
            }
        },
        permissions: true,
        questionsCount: true,
        score: true,
        stars: true,
        tags: () => relPartial(tagPartial, 'list'),
        transfersCount: true,
        views: true,
        you: () => relPartial(projectYouPartial, 'full'),
    },
    full: {
        versions: () => relPartial(projectVersionPartial, 'full', { omit: 'root' }),
        stats: () => relPartial(statsProjectPartial, 'full'),
    },
    list: {
        versions: () => relPartial(projectVersionPartial, 'list'),
    }
}