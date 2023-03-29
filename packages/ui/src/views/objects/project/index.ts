import { ProjectVersion, Session } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { projectVersionValidation } from '@shared/validation';
import { getCurrentUser } from 'utils/authentication/session';
import { getUserLanguages } from 'utils/display/translationTools';
import { validateAndGetYupErrors } from 'utils/shape/general';
import { ProjectVersionShape, shapeProjectVersion } from 'utils/shape/models/projectVersion';

export * from './ProjectCreate/ProjectCreate';
export * from './ProjectUpdate/ProjectUpdate';
export * from './ProjectView/ProjectView';

export const projectInitialValues = (
    session: Session | undefined,
    existing?: ProjectVersion | null | undefined
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

export const transformProjectValues = (o: ProjectVersionShape, u?: ProjectVersionShape) => {
    return u === undefined
        ? shapeProjectVersion.create(o)
        : shapeProjectVersion.update(o, u)
}

export const validateProjectValues = async (values: ProjectVersionShape, isCreate: boolean) => {
    const transformedValues = transformProjectValues(values);
    const validationSchema = isCreate
        ? projectVersionValidation.create({})
        : projectVersionValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}