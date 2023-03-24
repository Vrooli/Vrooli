import { DialogContent } from '@mui/material';
import { Resource, ResourceCreateInput, ResourceUpdateInput, ResourceUsedFor } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { resourceValidation } from '@shared/validation';
import { resourceCreate } from 'api/generated/endpoints/resource_create';
import { resourceUpdate } from 'api/generated/endpoints/resource_update';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { Formik } from 'formik';
import { BaseFormRef } from 'forms/BaseForm/BaseForm';
import { ResourceForm } from 'forms/ResourceForm/ResourceForm';
import { useCallback, useContext, useRef } from 'react';
import { getUserLanguages } from 'utils/display/translationTools';
import { PubSub } from 'utils/pubsub';
import { SessionContext } from 'utils/SessionContext';
import { ResourceShape, shapeResource } from 'utils/shape/models/resource';
import { ObjectSchema, ValidationError } from 'yup';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { LargeDialog } from '../LargeDialog/LargeDialog';
import { ResourceDialogProps } from '../types';

const helpText =
    `## What are resources?\n\nResources provide context to the object they are attached to, such as a  user, organization, project, or routine.\n\n## Examples\n**For a user** - Social media links, GitHub profile, Patreon\n\n**For an organization** - Official website, tools used by your team, news article explaining the vision\n\n**For a project** - Project Catalyst proposal, Donation wallet address\n\n**For a routine** - Guide, external service`

const titleId = "resource-dialog-title";

const isYupValidationError = (error: any): error is ValidationError => {
    return error.name === 'ValidationError';
}

const validateAndGetYupErrors = async (
    schema: ObjectSchema<any>,
    values: any
): Promise<{} | Record<string, string>> => {
    console.log('vagye a', schema, values)
    try {
        await schema.validate(values);
        console.log('vagye b')
        return {}; // Return an empty object if validation succeeds
    } catch (error) {
        console.log('vagye c', typeof error, (error as any).name);
        if (isYupValidationError(error)) {
            // Check if the inner array is not empty
            if (error.inner.length > 0) {
                // Convert the Yup error object to a Formik-compatible error object
                return error.inner.reduce((errors, err) => {
                    if (err && err.path) {
                        errors[err.path] = err.message;
                    }
                    return errors;
                }, {});
            } else if (error.path) {
                // Handle the case when the inner array is empty
                return { [error.path]: error.message };
            }
        }
        console.log('vagye e');
        // If it's not a Yup ValidationError, re-throw the error
        throw error;
    }
};

export const ResourceDialog = ({
    mutate,
    isOpen,
    onClose,
    onCreated,
    onUpdated,
    index,
    partialData,
    listId,
    zIndex,
}: ResourceDialogProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const [addMutation, { loading: addLoading }] = useCustomMutation<Resource, ResourceCreateInput>(resourceCreate);
    const [updateMutation, { loading: updateLoading }] = useCustomMutation<Resource, ResourceUpdateInput>(resourceUpdate);

    const handleClose = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Confirm dialog is dirty and closed by clicking outside
        console.log('ref current', formRef.current)
        formRef.current?.handleClose(onClose, reason !== 'backdropClick');
    }, [onClose]);

    const transformValues = useCallback((values: ResourceShape) => {
        console.log('transforming', values, index, shapeResource.create(values))
        return index < 0
            ? shapeResource.create(values)
            : shapeResource.update({ ...partialData, list: values.list } as ResourceShape, values)

    }, [index, partialData]);

    const validateFormValues = useCallback(
        async (values: ResourceShape) => {
            console.log('validating a', values, resourceValidation.create({}))
            const transformedValues = transformValues(values);
            console.log('validating b', transformedValues)
            const validationSchema = index < 0
                ? resourceValidation.create({})
                : resourceValidation.update({});
            console.log('validating c', validationSchema)
            const result = await validateAndGetYupErrors(validationSchema, transformedValues);
            console.log('validating d', result)
            return result;
        },
        [index, transformValues]
    );

    return (
        <>
            {/*  Main content */}
            <LargeDialog
                id="resource-dialog"
                onClose={handleClose}
                isOpen={isOpen}
                titleId={titleId}
                zIndex={zIndex}
            >
                <DialogTitle
                    id={titleId}
                    title={(index < 0) ? 'Add Resource' : 'Update Resource'}
                    helpText={helpText}
                    onClose={handleClose}
                />
                <DialogContent>
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
                            __typename: 'Resource' as const,
                            id: partialData?.id ?? DUMMY_ID,
                            index: partialData?.index ?? 0,
                            link: partialData?.link ?? '',
                            list: {
                                __typename: 'ResourceList' as const,
                                id: listId
                            },
                            usedFor: partialData?.usedFor ?? ResourceUsedFor.Context,
                            translations: partialData?.translations ?? [{
                                __typename: 'ResourceTranslation' as const,
                                id: DUMMY_ID,
                                language: getUserLanguages(session)[0],
                                description: '',
                                name: '',
                            }],
                        } as ResourceShape}
                        onSubmit={(values, helpers) => {
                            if (mutate) {
                                const onSuccess = (data: Resource) => {
                                    (index < 0) ? onCreated(data) : onUpdated(index ?? 0, data);
                                    helpers.resetForm();
                                    onClose();
                                }
                                console.log('yeeeet', index, values, shapeResource.create(values));
                                // If index is negative, create
                                const isCreating = index < 0;
                                if (!isCreating && (!partialData || !partialData.id)) {
                                    PubSub.get().publishSnack({ messageKey: 'ResourceNotFound', severity: 'Error' });
                                    return;
                                }
                                mutationWrapper<Resource, ResourceCreateInput | ResourceUpdateInput>({
                                    mutation: isCreating ? addMutation : updateMutation,
                                    input: transformValues(values),
                                    successMessage: () => ({ key: isCreating ? 'ResourceCreated' : 'ResourceUpdated' }),
                                    successCondition: (data) => data !== null,
                                    onSuccess,
                                    onError: () => { helpers.setSubmitting(false) },
                                })
                            } else {
                                onCreated({
                                    ...values,
                                    created_at: partialData?.created_at ?? new Date().toISOString(),
                                    updated_at: partialData?.updated_at ?? new Date().toISOString(),
                                } as Resource);
                                helpers.resetForm();
                                onClose();
                            }
                        }}
                        validate={async (values) => await validateFormValues(values)}
                    >
                        {(formik) => <ResourceForm
                            display="dialog"
                            isCreate={index < 0}
                            isLoading={addLoading || updateLoading}
                            isOpen={isOpen}
                            onCancel={handleClose}
                            ref={formRef}
                            zIndex={zIndex}
                            {...formik}
                        />}
                    </Formik>
                </DialogContent>
            </LargeDialog>
        </>
    )
}