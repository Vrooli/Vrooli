import { Session, SmartContractVersion } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { smartContractVersionValidation } from '@shared/validation';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { validateAndGetYupErrors } from 'utils/shape/general';
import { shapeSmartContractVersion, SmartContractVersionShape } from 'utils/shape/models/smartContractVersion';

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
    versionLabel: '1.0.0',
    ...existing,
});

export const transformSmartContractValues = (o: SmartContractVersionShape, u?: SmartContractVersionShape) => {
    return u === undefined
        ? shapeSmartContractVersion.create(o)
        : shapeSmartContractVersion.update(o, u)
}

export const validateSmartContractValues = async (values: SmartContractVersionShape, isCreate: boolean) => {
    const transformedValues = transformSmartContractValues(values);
    const validationSchema = isCreate
        ? smartContractVersionValidation.create({})
        : smartContractVersionValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}