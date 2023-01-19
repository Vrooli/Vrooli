import { ProjectVersionTranslation, ProjectVersionYou } from "@shared/consts";
import { GqlPartial } from "types";

export const projectTranslationPartial: GqlPartial<ProjectVersionTranslation> = {
    __typename: 'ProjectVersionTranslation',
    full: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
}

export const projectVersionYouPartial: GqlPartial<ProjectVersionYou> = {
    __typename: 'ProjectVersionYou',
    common: {
        canComment: true,
        canCopy: true,
        canDelete: true,
        canEdit: true,
        canReport: true,
        canUse: true,
        canView: true,
    },
    full: {
        runs: runProjectPartial.full,
    },
    list: {
        runs: runProjectPartial.list,
    }
}

export const listProjectVersionFields = ['ProjectVersion', `{
    id
}`] as const;
export const projectVersionFields = ['ProjectVersion', `{
    id
}`] as const;