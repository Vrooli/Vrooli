import { ApiVersion, Session } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { apiVersionValidation } from '@shared/validation';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { validateAndGetYupErrors } from 'utils/shape/general';
import { ApiVersionShape, shapeApiVersion } from 'utils/shape/models/apiVersion';

export * from './ApiCreate/ApiCreate';
export * from './ApiUpdate/ApiUpdate';
export * from './ApiView/ApiView';

export const apiInitialValues = (
    session: Session | undefined,
    existing?: ApiVersion | null | undefined
): ApiVersionShape => ({
    __typename: 'ApiVersion' as const,
    id: DUMMY_ID,
    callLink: '',
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'Api' as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
        tags: [],
    },
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        details: '',
        name: '',
        summary: '',
    }],
    versionLabel: '1.0.0',
    ...existing,
});

export const transformApiValues = (o: ApiVersionShape, u?: ApiVersionShape) => {
    return u === undefined
        ? shapeApiVersion.create(o)
        : shapeApiVersion.update(o, u)
}

export const validateApiValues = async (values: ApiVersionShape, isCreate: boolean) => {
    const transformedValues = transformApiValues(values);
    const validationSchema = isCreate
        ? apiVersionValidation.create({})
        : apiVersionValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}