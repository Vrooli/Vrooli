import { DeleteOneInput, DeleteType, endpointGetReminder, endpointPostDeleteOne, endpointPostReminder, endpointPutReminder, Reminder, ReminderCreateInput, ReminderUpdateInput, Success } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderForm, reminderInitialValues, transformReminderValues, validateReminderValues } from "forms/ReminderForm/ReminderForm";
import { DeleteIcon } from "icons";
import { useCallback, useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { tryOnClose } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ReminderShape, shapeReminder } from "utils/shape/models/reminder";
import { ReminderViewProps } from "../types";

export const ReminderView = ({
    onClose,
    overrideObject,
    zIndex,
}: ReminderViewProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { display, isLoading: isReadLoading, object: existing } = useObjectFromUrl<Reminder, ReminderShape>({
        ...endpointGetReminder,
        objectType: "Reminder",
        overrideObject,
        transform: (data) => reminderInitialValues(session, data),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetchCreate,
        fetchUpdate,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Reminder, ReminderCreateInput, ReminderUpdateInput>({
        display,
        endpointCreate: endpointPostReminder,
        endpointUpdate: endpointPutReminder,
        isCreate: false,
    });

    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const handleDelete = useCallback((id: string) => {
        if (!existing) return;
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
                        inputs: shapeReminder.create({
                            ...existing,
                            // Make sure not to set any extra fields, 
                            // so this is treated as a "Connect" instead of a "Create"
                            reminderList: {
                                __typename: "ReminderList" as const,
                                id: existing.reminderList.id,
                            },
                        }),
                        successCondition: (data) => !!data.id,
                    });
                },
            }),
            onSuccess: () => {
                tryOnClose(onClose, setLocation);
            },
            errorMessage: () => ({ messageKey: "FailedToDelete" }),
        });
    }, [deleteMutation, existing, fetchCreate, onClose, setLocation]);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t("Reminder")}
                options={[{
                    Icon: DeleteIcon,
                    label: t("Delete"),
                    onClick: handleDelete,
                }]}
                zIndex={zIndex}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<ReminderUpdateInput, Reminder>({
                        fetch: fetchUpdate,
                        inputs: transformReminderValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateReminderValues(values, existing)}
            >
                {(formik) => <ReminderForm
                    display={display}
                    isCreate={false}
                    isLoading={isCreateLoading || isDeleteLoading || isReadLoading || isUpdateLoading}
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
