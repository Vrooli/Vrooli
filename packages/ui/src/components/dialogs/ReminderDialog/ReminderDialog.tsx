import { Reminder } from '@shared/consts';
import { Formik } from 'formik';
import { BaseFormRef } from 'forms/BaseForm/BaseForm';
import { ReminderForm, reminderInitialValues, validateReminderValues } from 'forms/ReminderForm.tsx/ReminderForm';
import { useCallback, useContext, useMemo, useRef } from 'react';
import { SessionContext } from 'utils/SessionContext';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { LargeDialog } from '../LargeDialog/LargeDialog';
import { ReminderDialogProps } from '../types';

const titleId = "reminder-dialog-title";

export const ReminderDialog = ({
    isOpen,
    onClose,
    onCreated,
    onUpdated,
    index,
    partialData,
    listId,
    zIndex,
}: ReminderDialogProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => reminderInitialValues(session, listId, partialData as any), [listId, partialData, session]);

    const handleClose = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Confirm dialog is dirty and closed by clicking outside
        formRef.current?.handleClose(onClose, reason !== 'backdropClick');
    }, [onClose]);

    return (
        <>
            {/*  Main content */}
            <LargeDialog
                id="reminder-dialog"
                onClose={handleClose}
                isOpen={isOpen}
                titleId={titleId}
                zIndex={zIndex}
            >
                <DialogTitle
                    id={titleId}
                    title={(index < 0) ? 'Add Reminder' : 'Update Reminder'}
                    onClose={handleClose}
                />
                <Formik
                    enableReinitialize={true}
                    initialValues={initialValues}
                    onSubmit={(values, helpers) => {
                        const reminder = {
                            ...values,
                            created_at: partialData?.created_at ?? new Date().toISOString(),
                            updated_at: partialData?.updated_at ?? new Date().toISOString(),
                        } as Reminder
                        if (index < 0) onCreated(reminder);
                        else onUpdated(index ?? 0, reminder);
                        helpers.resetForm();
                        onClose();
                    }}
                    validate={async (values) => await validateReminderValues(values, partialData as any)}
                >
                    {(formik) => <ReminderForm
                        display="dialog"
                        isCreate={index < 0}
                        isLoading={false}
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