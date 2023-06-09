import { DeleteIcon, endpointGetReminder, endpointPostReminder, endpointPutReminder, FindByIdInput, Reminder, ReminderCreateInput, ReminderUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderForm, reminderInitialValues, transformReminderValues, validateReminderValues } from "forms/ReminderForm/ReminderForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ReminderUpsertProps } from "../types";

export const ReminderUpsert = ({
    display = "page",
    handleDelete,
    isCreate,
    listId,
    onCancel,
    onCompleted,
    partialData,
    zIndex = 200,
}: ReminderUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    // Fetch existing data
    const id = useMemo(() => isCreate ? undefined : (partialData?.id ?? parseSingleItemUrl()?.id), [isCreate, partialData?.id]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindByIdInput, Reminder>(endpointGetReminder);
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => reminderInitialValues(session, listId, { ...existing, ...partialData } as Reminder), [existing, listId, partialData, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<Reminder>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<ReminderCreateInput, Reminder>(endpointPostReminder);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<ReminderUpdateInput, Reminder>(endpointPutReminder);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<ReminderCreateInput | ReminderUpdateInput, Reminder>;

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
                    onClick: handleDelete,
                }] : []}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
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
