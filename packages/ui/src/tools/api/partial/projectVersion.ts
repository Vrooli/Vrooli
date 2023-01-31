import { ProjectVersion, ProjectVersionTranslation, ProjectVersionYou } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";
import { versionYouPartial } from "./root";

export const projectVersionTranslationPartial: GqlPartial<ProjectVersionTranslation> = {
    __typename: 'ProjectVersionTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
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
        runs: async () => relPartial((await import('./runProject')).runProjectPartial, 'full', { omit: 'projectVersion' }),
    },
    list: {
        runs: async () => relPartial((await import('./runProject')).runProjectPartial, 'list', { omit: 'projectVersion' }),
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
        directories: async () => relPartial((await import('./projectVersionDirectory')).projectVersionDirectoryPartial, 'full', { omit: 'projectVersion ' }),
        pullRequest: async () => relPartial((await import('./pullRequest')).pullRequestPartial, 'full'),
        root: async () => relPartial((await import('./project')).projectPartial, 'full', { omit: 'versions' }),
        translations: () => relPartial(projectVersionTranslationPartial, 'full'),
        versionNotes: true,
    },
    list: {
        directories: async () => relPartial((await import('./projectVersionDirectory')).projectVersionDirectoryPartial, 'list', { omit: 'projectVersion ' }),
        root: async () => relPartial((await import('./project')).projectPartial, 'list', { omit: 'versions' }),
        translations: () => relPartial(projectVersionTranslationPartial, 'list'),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => relPartial((await import('./project')).projectPartial, 'nav'),
        translations: () => relPartial(projectVersionTranslationPartial, 'list'),
    }
}