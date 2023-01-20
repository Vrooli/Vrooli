import { ProjectVersion, ProjectVersionTranslation, ProjectVersionYou } from "@shared/consts";
import { relPartial } from "graphql/utils";
import { GqlPartial } from "types";
import { projectPartial } from "./project";
import { pullRequestPartial } from "./pullRequest";
import { versionYouPartial } from "./root";

export const projectVersionTranslationPartial: GqlPartial<ProjectVersionTranslation> = {
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
    },
}

export const projectVersionPartial: GqlPartial<ProjectVersion> = {
    __typename: 'ProjectVersion',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        directoriesCount: true,
        isLatest: true,
        isPrivate: true,
        reportsCount: true,
        runsCount: true,
        simplicity: true,
        versionIndex: true,
        versionLabel: true,
        you: () => relPartial(versionYouPartial, 'full'),
    },
    full: {
        directories: () => relPartial(projectVersionDirectoryPartial, 'full'),
        pullRequest: () => relPartial(pullRequestPartial, 'full'),
        root: () => relPartial(projectPartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(projectVersionTranslationPartial, 'full'),
        versionNotes: true,
    },
    list: {
        directories: () => relPartial(projectVersionDirectoryPartial, 'list'),
        root: () => relPartial(projectPartial, 'list', { omit: 'versions' }),
        translations: () => relPartial(projectVersionTranslationPartial, 'list'),
    }
}