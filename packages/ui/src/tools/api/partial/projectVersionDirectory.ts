import { ProjectVersionDirectory, ProjectVersionDirectoryTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "../types";

export const projectVersionDirectoryTranslationPartial: GqlPartial<ProjectVersionDirectoryTranslation> = {
    __typename: 'ProjectVersionDirectoryTranslation',
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
    full: {},
    list: {},
}

export const projectVersionDirectoryPartial: GqlPartial<ProjectVersionDirectory> = {
    __typename: 'ProjectVersionDirectory',
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        childOrder: true,
        isRoot: true,
        projectVersion: async () => relPartial((await import('./projectVersion')).projectVersionPartial, 'nav', { omit: 'directories' }),
    },
    full: {
        children: () => relPartial(projectVersionDirectoryPartial, 'nav', { omit: ['parentDirectory', 'children'] }),
        childApiVersions: async () => relPartial((await import('./apiVersion')).apiVersion, 'nav'),
        childNoteVersions: async () => relPartial((await import('./noteVersion')).noteVersionPartial, 'nav'),
        childOrganizations: async () => relPartial((await import('./organization')).organizationPartial, 'nav'),
        childProjectVersions: async () => relPartial((await import('./projectVersion')).projectVersionPartial, 'nav'),
        childRoutineVersions: async () => relPartial((await import('./routineVersion')).routineVersionPartial, 'nav'),
        childSmartContractVersions: async () => relPartial((await import('./smartContractVersion')).smartContractVersionPartial, 'nav'),
        childStandardVersions: async () => relPartial((await import('./standardVersion')).standardVersionPartial, 'nav'),
        parentDirectory: () => relPartial(projectVersionDirectoryPartial, 'nav', { omit: ['parentDirectory', 'children'] }),
        translations: () => relPartial(projectVersionDirectoryTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(projectVersionDirectoryTranslationPartial, 'list'),
    }
}