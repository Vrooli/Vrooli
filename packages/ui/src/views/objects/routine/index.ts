import { RoutineVersion, Session } from '@shared/consts';
import { DUMMY_ID, uuid } from '@shared/uuid';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { RoutineVersionShape } from 'utils/shape/models/routineVersion';

export * from './RoutineCreate/RoutineCreate';
export * from './RoutineUpdate/RoutineUpdate';
export * from './RoutineView/RoutineView';

export const routineInitialValues = (
    session: Session | undefined,
    existing?: RoutineVersion | null | undefined
): RoutineVersionShape => ({
    __typename: 'RoutineVersion' as const,
    id: uuid(), // Cannot be a dummy ID because nodes, links, etc. reference this ID
    inputs: [],
    isComplete: false,
    isPrivate: false,
    directoryListings: [],
    nodeLinks: [],
    nodes: [],
    outputs: [],
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