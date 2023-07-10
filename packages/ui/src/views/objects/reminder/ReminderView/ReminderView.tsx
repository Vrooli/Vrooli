import { DeleteIcon, DeleteOneInput, DeleteType, endpointGetReminder, endpointPostDeleteOne, endpointPostReminder, endpointPutReminder, FindByIdInput, Reminder, ReminderCreateInput, ReminderUpdateInput, Success, useLocation } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderForm, reminderInitialValues, transformReminderValues, validateReminderValues } from "forms/ReminderForm/ReminderForm";
import { useCallback, useContext, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl, tryOnClose } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeReminder } from "utils/shape/models/reminder";
import { ReminderViewProps } from "../types";

export const ReminderView = ({
    display = "page",
    onClose,
    partialData,
    zIndex,
}: ReminderViewProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    // Fetch existing data
    const id = useMemo(() => partialData?.id ?? parseSingleItemUrl({})?.id, [partialData?.id]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindByIdInput, Reminder>(endpointGetReminder);
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => reminderInitialValues(session, existing?.reminderList?.id ?? "", { ...existing, ...partialData } as Reminder), [existing, partialData, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<Reminder>(display, false);
    const [create, { loading: isCreateLoading }] = useLazyFetch<ReminderCreateInput, Reminder>(endpointPostReminder);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<ReminderUpdateInput, Reminder>(endpointPutReminder);

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
                        fetch: create,
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
    }, [create, deleteMutation, existing, onClose, setLocation]);

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
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<ReminderUpdateInput, Reminder>({
                        fetch: update,
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
