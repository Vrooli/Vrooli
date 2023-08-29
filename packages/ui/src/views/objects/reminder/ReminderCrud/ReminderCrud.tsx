import { endpointGetReminder, endpointPostReminder, endpointPutReminder, Reminder, ReminderCreateInput, ReminderUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { ReminderForm, reminderInitialValues, transformReminderValues, validateReminderValues } from "forms/ReminderForm/ReminderForm";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext } from "react";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { ReminderShape } from "utils/shape/models/reminder";
import { ReminderCrudProps } from "../types";

export const ReminderCrud = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    onDeleted,
    overrideObject,
}: ReminderCrudProps) => {
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Reminder, ReminderShape>({
        ...endpointGetReminder,
        objectType: "Reminder",
        overrideObject: overrideObject as Reminder,
        transform: (existing) => reminderInitialValues(session, existing),
    });

    const {
        fetch,
        fetchCreate,
        handleCancel,
        handleCreated,
        handleCompleted,
        handleDeleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Reminder, ReminderCreateInput, ReminderUpdateInput>({
        display,
        endpointCreate: endpointPostReminder,
        endpointUpdate: endpointPutReminder,
        isCreate,
        onCancel,
        onCompleted,
        onDeleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="reminder-crud-dialog"
            isOpen={isOpen ?? false}
            onClose={handleClose}
        >
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<ReminderCreateInput | ReminderUpdateInput, Reminder>({
                        fetch,
                        inputs: transformReminderValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateReminderValues(values, existing, isCreate)}
            >
                {(formik) => <ReminderForm
                    display={display}
                    fetchCreate={fetchCreate}
                    handleClose={handleClose}
                    handleCreated={handleCreated}
                    handleDeleted={handleDeleted}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
