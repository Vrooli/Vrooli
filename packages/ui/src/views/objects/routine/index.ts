import { RoutineVersion, Session } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { RoutineVersionShape } from 'utils/shape/models/routineVersion';

export * from './RoutineCreate/RoutineCreate';
export * from './RoutineUpdate/RoutineUpdate';
export * from './RoutineView/RoutineView';

export const routineInitialValues = (
    session: Session | undefined,
    existing?: RoutineVersion | undefined
): RoutineVersionShape => ({
    __typename: 'RoutineVersion' as const,
    id: DUMMY_ID,
    isComplete: false,
    isPrivate: false,
    directoryListings: [],
    nodeLinks: [],
    nodes: [],
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'Routine' as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
        parent: null,
        tags: [],
    },
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: '',
        instructions: '',
        name: '',
    }],
    versionLabel: '1.0.0',
    ...existing,
});