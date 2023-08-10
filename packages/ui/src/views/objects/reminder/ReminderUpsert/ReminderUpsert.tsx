import { endpointGetReminder, endpointPostReminder, endpointPutReminder, Reminder, ReminderCreateInput, ReminderUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderForm, reminderInitialValues, transformReminderValues, validateReminderValues } from "forms/ReminderForm/ReminderForm";
import { DeleteIcon } from "icons";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ReminderShape } from "utils/shape/models/reminder";
import { ReminderUpsertProps } from "../types";

export const ReminderUpsert = ({
    handleDelete,
    isCreate,
    listId,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: ReminderUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const { display, isLoading: isReadLoading, object: existing } = useObjectFromUrl<Reminder, ReminderShape>({
        ...endpointGetReminder,
        objectType: "Reminder",
        overrideObject,
        transform: (existing) => reminderInitialValues(session, listId, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Reminder, ReminderCreateInput, ReminderUpdateInput>({
        display,
        endpointCreate: endpointPostReminder,
        endpointUpdate: endpointPutReminder,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateReminder" : "UpdateReminder")}
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
                        inputs: transformReminderValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateReminderValues(values, existing)}
            >
                {(formik) => <ReminderForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    );
};
