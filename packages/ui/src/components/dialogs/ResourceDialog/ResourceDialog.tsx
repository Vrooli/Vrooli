import { DialogContent } from '@mui/material';
import { Resource, ResourceCreateInput, ResourceList, ResourceUpdateInput, ResourceUsedFor } from '@shared/consts';
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
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { LargeDialog } from '../LargeDialog/LargeDialog';
import { ResourceDialogProps } from '../types';

const helpText =
    `## What are resources?\n\nResources provide context to the object they are attached to, such as a  user, organization, project, or routine.\n\n## Examples\n**For a user** - Social media links, GitHub profile, Patreon\n\n**For an organization** - Official website, tools used by your team, news article explaining the vision\n\n**For a project** - Project Catalyst proposal, Donation wallet address\n\n**For a routine** - Guide, external service`

const titleId = "resource-dialog-title";

export const ResourceDialog = ({
    mutate,
    open,
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

    return (
        <>
            {/*  Main content */}
            <LargeDialog
                id="resource-dialog"
                onClose={handleClose}
                isOpen={open}
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
                            index: partialData?.index ?? Math.max(index, 0),
                            link: partialData?.link ?? '',
                            listConnect: listId,
                            usedFor: partialData?.usedFor ?? ResourceUsedFor.Context,
                            translations: partialData?.translations ?? [{
                                __typename: 'ResourceTranslation' as const,
                                id: DUMMY_ID,
                                language: getUserLanguages(session)[0],
                                description: '',
                                name: '',
                            }],
                        }}
                        onSubmit={(values, helpers) => {
                            const input = {
                                ...values,
                                list: {
                                    __typename: 'ResourceList' as const,
                                    id: values.listConnect,
                                } as ResourceList,
                            };
                            if (mutate) {
                                const onSuccess = (data: Resource) => {
                                    (index < 0) ? onCreated(data) : onUpdated(index ?? 0, data);
                                    helpers.resetForm();
                                    onClose();
                                }
                                // If index is negative, create
                                if (index < 0) {
                                    mutationWrapper<Resource, ResourceCreateInput>({
                                        mutation: addMutation,
                                        input: shapeResource.create(input),
                                        successMessage: () => ({ key: 'ResourceCreated' }),
                                        successCondition: (data) => data !== null,
                                        onSuccess,
                                        onError: () => { helpers.setSubmitting(false) },
                                    })
                                }
                                // Otherwise, update
                                else {
                                    if (!partialData || !partialData.id) {
                                        PubSub.get().publishSnack({ messageKey: 'ResourceNotFound', severity: 'Error' });
                                        return;
                                    }
                                    mutationWrapper<Resource, ResourceUpdateInput>({
                                        mutation: updateMutation,
                                        input: shapeResource.update({ ...partialData, list: { id: listId } } as ResourceShape, input),
                                        successMessage: () => ({ key: 'ResourceUpdated' }),
                                        successCondition: (data) => data !== null,
                                        onSuccess,
                                        onError: () => { helpers.setSubmitting(false) },
                                    })
                                }
                            } else {
                                onCreated({
                                    ...input,
                                    created_at: partialData?.created_at ?? new Date().toISOString(),
                                    updated_at: partialData?.updated_at ?? new Date().toISOString(),
                                });
                                helpers.resetForm();
                                onClose();
                            }
                        }}
                        validationSchema={index < 0 ? resourceValidation.create({}) : resourceValidation.update({})}
                    >
                        {(formik) => <ResourceForm
                            display="dialog"
                            index={index}
                            isLoading={addLoading || updateLoading}
                            onCancel={handleClose}
                            open={open}
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