import { ProjectVersionDirectory, ProjectVersionDirectoryTranslation } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

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
        projectVersion: () => relPartial(require('./projectVersion').projectVersionPartial, 'nav', { omit: 'directories' }),
    },
    full: {
        children: () => relPartial(projectVersionDirectoryPartial, 'nav', { omit: ['parentDirectory', 'children'] }),
        childApiVersions: () => relPartial(require('./apiVersion').apiVersionPartial, 'nav'),
        childNoteVersions: () => relPartial(require('./noteVersion').noteVersionPartial, 'nav'),
        childOrganizations: () => relPartial(require('./organization').organizationPartial, 'nav'),
        childProjectVersions: () => relPartial(require('./projectVersion').projectVersionPartial, 'nav'),
        childRoutineVersions: () => relPartial(require('./routineVersion').routineVersionPartial, 'nav'),
        childSmartContractVersions: () => relPartial(require('./smartContractVersion').smartContractVersionPartial, 'nav'),
        childStandardVersions: () => relPartial(require('./standardVersion').standardVersionPartial, 'nav'),
        parentDirectory: () => relPartial(projectVersionDirectoryPartial, 'nav', { omit: ['parentDirectory', 'children'] }),
        translations: () => relPartial(projectVersionDirectoryTranslationPartial, 'full'),
    },
    list: {
        translations: () => relPartial(projectVersionDirectoryTranslationPartial, 'list'),
    }
}