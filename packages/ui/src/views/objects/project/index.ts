import { ProjectVersion, Session } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { ProjectVersionShape } from 'utils/shape/models/projectVersion';

export * from './ProjectCreate/ProjectCreate';
export * from './ProjectUpdate/ProjectUpdate';
export * from './ProjectView/ProjectView';

export const projectInitialValues = (
    session: Session | undefined,
    existing?: ProjectVersion | undefined
): ProjectVersionShape => ({
    __typename: 'ProjectVersion' as const,
    id: DUMMY_ID,
    directoryListings: [],
    isComplete: false,
    isPrivate: true,
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'Project' as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
        parent: null,
        tags: [],
    },
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        name: '',
        description: '',
    }],
    versionLabel: '1.0.0',
    ...existing,
});