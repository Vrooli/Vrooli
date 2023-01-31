import { ProjectVersion, ProjectVersionTranslation, ProjectVersionYou } from "@shared/consts";
import { rel } from '../utils';
import { GqlPartial } from "../types";
import { versionYou } from "./root";

export const projectVersionTranslation: GqlPartial<ProjectVersionTranslation> = {
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

export const projectVersionYou: GqlPartial<ProjectVersionYou> = {
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
        runs: async () => rel((await import('./runProject')).runProject, 'full', { omit: 'projectVersion' }),
    },
    list: {
        runs: async () => rel((await import('./runProject')).runProject, 'list', { omit: 'projectVersion' }),
    },
}

export const projectVersion: GqlPartial<ProjectVersion> = {
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
        you: () => rel(versionYou, 'full'),
    },
    full: {
        directories: async () => rel((await import('./projectVersionDirectory')).projectVersionDirectory, 'full', { omit: 'projectVersion ' }),
        pullRequest: async () => rel((await import('./pullRequest')).pullRequest, 'full'),
        root: async () => rel((await import('./project')).project, 'full', { omit: 'versions' }),
        translations: () => rel(projectVersionTranslation, 'full'),
        versionNotes: true,
    },
    list: {
        directories: async () => rel((await import('./projectVersionDirectory')).projectVersionDirectory, 'list', { omit: 'projectVersion ' }),
        root: async () => rel((await import('./project')).project, 'list', { omit: 'versions' }),
        translations: () => rel(projectVersionTranslation, 'list'),
    },
    nav: {
        id: true,
        isLatest: true,
        isPrivate: true,
        versionIndex: true,
        versionLabel: true,
        root: async () => rel((await import('./project')).project, 'nav'),
        translations: () => rel(projectVersionTranslation, 'list'),
    }
}