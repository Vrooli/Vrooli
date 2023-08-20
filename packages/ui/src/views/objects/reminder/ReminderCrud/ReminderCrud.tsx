import { DeleteOneInput, DeleteType, endpointGetReminder, endpointPostDeleteOne, endpointPostReminder, endpointPutReminder, Reminder, ReminderCreateInput, ReminderUpdateInput, Success } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderForm, reminderInitialValues, transformReminderValues, validateReminderValues } from "forms/ReminderForm/ReminderForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { DeleteIcon } from "icons";
import { useCallback, useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { firstString } from "utils/display/stringTools";
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
    zIndex,
}: ReminderCrudProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Reminder, ReminderShape>({
        ...endpointGetReminder,
        objectType: "Reminder",
        overrideObject: overrideObject as Reminder,
        transform: (existing) => reminderInitialValues(session, existing),
    });
    console.log("reminderUpsert render", existing);

    const formRef = useRef<BaseFormRef>();
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

    // Handle delete
    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const handleDelete = useCallback((id: string) => {
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteMutation,
            inputs: { id: existing.id, objectType: DeleteType.Reminder },
            successCondition: (data) => data.success,
            successMessage: () => ({
                messageKey: "ObjectDeleted",
                messageVariables: { objectName: existing.name },
                buttonKey: "Undo",
                buttonClicked: () => {
                    fetchLazyWrapper<ReminderCreateInput, Reminder>({
                        fetch: fetchCreate,
                        inputs: transformReminderValues({
                            ...existing,
                            // Make sure not to set any extra fields, 
                            // so this is treated as a "Connect" instead of a "Create"
                            reminderList: {
                                __typename: "ReminderList" as const,
                                id: existing.reminderList.id,
                            },
                        }, existing, true) as ReminderCreateInput,
                        successCondition: (data) => !!data.id,
                        onSuccess: (data) => { handleCreated(data); },
                    });
                },
            }),
            onSuccess: () => {
                handleDeleted(existing as Reminder);
            },
            errorMessage: () => ({ messageKey: "FailedToDelete" }),
        });
    }, [deleteMutation, existing, fetchCreate, handleCreated, handleDeleted]);

    return (
        <MaybeLargeDialog
            display={display}
            id="reminder-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
            zIndex={zIndex}
        >
            <TopBar
                display={display}
                onClose={handleCancel}
                title={firstString(getDisplay(existing).title, t(isCreate ? "CreateReminder" : "UpdateReminder"))}
                // Show delete button only when updating
                options={!isCreate ? [{
                    Icon: DeleteIcon,
                    label: t("Delete"),
                    onClick: handleDelete as () => void,
                }] : []}
                zIndex={zIndex}
            />
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
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading || isDeleteLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
