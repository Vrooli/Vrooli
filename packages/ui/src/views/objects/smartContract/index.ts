import { Session, SmartContractVersion } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { SmartContractVersionShape } from 'utils/shape/models/smartContractVersion';

export * from './SmartContractCreate/SmartContractCreate';
export * from './SmartContractUpdate/SmartContractUpdate';
export * from './SmartContractView/SmartContractView';

export const smartContractInitialValues = (
    session: Session | undefined,
    existing?: SmartContractVersion | undefined
): SmartContractVersionShape => ({
    __typename: 'SmartContractVersion' as const,
    id: DUMMY_ID,
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    content: '',
    contractType: '',
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'SmartContract' as const,
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
        jsonVariable: '',
        name: '',
    }],
    ...existing,
});