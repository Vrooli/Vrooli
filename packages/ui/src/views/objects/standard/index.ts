import { Session, StandardVersion } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { standardVersionValidation } from '@shared/validation';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { validateAndGetYupErrors } from 'utils/shape/general';
import { shapeStandardVersion, StandardVersionShape } from 'utils/shape/models/standardVersion';

export * from './StandardCreate/StandardCreate';
export * from './StandardUpdate/StandardUpdate';
export * from './StandardView/StandardView';

export const standardInitialValues = (
    session: Session | undefined,
    existing?: StandardVersion | null | undefined
): StandardVersionShape => ({
    __typename: 'StandardVersion' as const,
    id: DUMMY_ID,
    default: '',
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    isFile: false,
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'Standard' as const,
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
        jsonVariable: null, //TODO
        name: '',
    }],
    versionLabel: '1.0.0',
    ...existing,
});

export const transformStandardValues = (o: StandardVersionShape, u?: StandardVersionShape) => {
    return u === undefined
        ? shapeStandardVersion.create(o)
        : shapeStandardVersion.update(o, u)
}

export const validateStandardValues = async (values: StandardVersionShape, isCreate: boolean) => {
    const transformedValues = transformStandardValues(values);
    const validationSchema = isCreate
        ? standardVersionValidation.create({})
        : standardVersionValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}