import { FocusMode, FocusModeCreateInput, FocusModeUpdateInput } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { focusModeValidation } from '@shared/validation';
import { focusModeCreate } from 'api/generated/endpoints/focusMode_create';
import { focusModeUpdate } from 'api/generated/endpoints/focusMode_update';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { Formik } from 'formik';
import { BaseFormRef } from 'forms/BaseForm/BaseForm';
import { FocusModeForm } from 'forms/FocusModeForm/FocusModeForm';
import { useCallback, useRef } from 'react';
import { PubSub } from 'utils/pubsub';
import { validateAndGetYupErrors } from 'utils/shape/general';
import { FocusModeShape, shapeFocusMode } from 'utils/shape/models/focusMode';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { LargeDialog } from '../LargeDialog/LargeDialog';
import { FocusModeDialogProps } from '../types';

const titleId = "focus-mode-dialog-title";

export const FocusModeDialog = ({
    isCreate,
    isOpen,
    onClose,
    onCreated,
    onUpdated,
    partialData,
    zIndex,
}: FocusModeDialogProps) => {
    const formRef = useRef<BaseFormRef>();
    const [addMutation, { loading: addLoading }] = useCustomMutation<FocusMode, FocusModeCreateInput>(focusModeCreate);
    const [updateMutation, { loading: updateLoading }] = useCustomMutation<FocusMode, FocusModeUpdateInput>(focusModeUpdate);

    const handleClose = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Confirm dialog is dirty and closed by clicking outside
        formRef.current?.handleClose(onClose, reason !== 'backdropClick');
    }, [onClose]);

    const transformValues = useCallback((values: FocusModeShape) => {
        return isCreate
            ? shapeFocusMode.create(values)
            : shapeFocusMode.update(partialData as any, values)

    }, [isCreate, partialData]);

    const validateFormValues = useCallback(
        async (values: FocusModeShape) => {
            console.log('validating a', values, focusModeValidation.create({}))
            const transformedValues = transformValues(values);
            console.log('validating b', transformedValues)
            const validationSchema = isCreate
                ? focusModeValidation.create({})
                : focusModeValidation.update({});
            console.log('validating c', validationSchema)
            const result = await validateAndGetYupErrors(validationSchema, transformedValues);
            console.log('validating d', result)
            return result;
        },
        [isCreate, transformValues]
    );

    return (
        <>
            {/*  Main content */}
            <LargeDialog
                id="focus-mode-dialog"
                onClose={handleClose}
                isOpen={isOpen}
                titleId={titleId}
                zIndex={zIndex}
            >
                <DialogTitle
                    id={titleId}
                    title={(isCreate) ? 'Add Focus Mode' : 'Update Focus Mode'}
                    onClose={handleClose}
                />
                <Formik
                    enableReinitialize={true}
                    initialValues={{
                        __typename: 'FocusMode' as const,
                        id: DUMMY_ID,
                        description: '',
                        name: '',
                        reminderList: {
                            __typename: 'ReminderList' as const,
                            id: DUMMY_ID,
                            reminders: [],
                        },
                        resourceList: {
                            __typename: 'ResourceList' as const,
                            id: DUMMY_ID,
                            resources: [],
                        },
                        filters: [],
                        schedule: null,
                        ...partialData,
                    } as FocusModeShape}
                    onSubmit={(values, helpers) => {
                        const onSuccess = (data: FocusMode) => {
                            isCreate ? onCreated(data) : onUpdated(data);
                            helpers.resetForm();
                            onClose();
                        }
                        console.log('yeeeet', values, shapeFocusMode.create(values));
                        // If index is negative, create
                        const isCreating = isCreate;
                        if (!isCreating && (!partialData || !partialData.id)) {
                            PubSub.get().publishSnack({ messageKey: 'NotFound', severity: 'Error' });
                            return;
                        }
                        mutationWrapper<FocusMode, FocusModeCreateInput | FocusModeUpdateInput>({
                            mutation: isCreating ? addMutation : updateMutation,
                            input: transformValues(values),
                            successMessage: () => ({ key: isCreating ? 'FocusModeCreated' : 'FocusModeUpdated' }),
                            successCondition: (data) => data !== null,
                            onSuccess,
                            onError: () => { helpers.setSubmitting(false) },
                        })
                    }}
                    validate={async (values) => await validateFormValues(values)}
                >
                    {(formik) => <FocusModeForm
                        display="dialog"
                        isCreate={isCreate}
                        isLoading={addLoading || updateLoading}
                        isOpen={isOpen}
                        onCancel={handleClose}
                        ref={formRef}
                        zIndex={zIndex}
                        {...formik}
                    />}
                </Formik>
            </LargeDialog>
        </>
    )
}