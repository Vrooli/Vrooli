import { DeleteIcon, FindByIdInput, Reminder, ReminderCreateInput, ReminderUpdateInput } from "@local/shared";
import { Box, Button } from "@mui/material";
import { mutationWrapper } from "api";
import { reminderCreate } from "api/generated/endpoints/reminder_create";
import { reminderFindOne } from "api/generated/endpoints/reminder_findOne";
import { reminderUpdate } from "api/generated/endpoints/reminder_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderForm, reminderInitialValues, transformReminderValues, validateReminderValues } from "forms/ReminderForm/ReminderForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
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
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<Reminder, FindByIdInput>(reminderFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => reminderInitialValues(session, listId, { ...existing, ...partialData } as Reminder), [existing, listId, partialData, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<Reminder>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<Reminder, ReminderCreateInput>(reminderCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<Reminder, ReminderUpdateInput>(reminderUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateReminder" : "UpdateReminder",
                }}
                // Show delete button only when updating
                below={
                    !isCreate ? (
                        <Box pb={2} sx={{ display: "flex", justifyContent: "center" }}>
                            <Button
                                onClick={handleDelete}
                                startIcon={<DeleteIcon />}
                            >{t("Delete")}</Button>
                        </Box>
                    ) : undefined
                }
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper<Reminder, ReminderCreateInput | ReminderUpdateInput>({
                        mutation,
                        input: transformReminderValues(values, existing),
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
