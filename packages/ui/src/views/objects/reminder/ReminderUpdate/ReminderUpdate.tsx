import { FindByIdInput, Reminder, ReminderUpdateInput } from "@shared/consts";
import { uuid } from '@shared/uuid';
import { reminderValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { reminderFindOne } from "api/generated/endpoints/reminder_findOne";
import { reminderUpdate } from "api/generated/endpoints/reminder_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderForm } from "forms/ReminderForm.tsx/ReminderForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeReminder } from "utils/shape/models/reminder";
import { ReminderUpdateProps } from "../types";

export const ReminderUpdate = ({
    display = 'page',
    zIndex = 200,
}: ReminderUpdateProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: reminder, loading: isReadLoading }] = useCustomLazyQuery<Reminder, FindByIdInput>(reminderFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const { onCancel, onUpdated } = useUpdateActions<Reminder>();
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<Reminder, ReminderUpdateInput>(reminderUpdate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'UpdateReminder',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    __typename: 'Reminder' as const,
                    id: uuid(),
                }}
                onSubmit={(values, helpers) => {
                    if (!reminder) {
                        PubSub.get().publishSnack({ messageKey: 'CouldNotReadReminder', severity: 'Error' });
                        return;
                    }
                    mutationWrapper<Reminder, ReminderUpdateInput>({
                        mutation,
                        input: shapeReminder.update(reminder, values),
                        onSuccess: (data) => { onUpdated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={reminderValidation.update({})}
            >
                {(formik) => <ReminderForm
                    display={display}
                    isCreate={false}
                    isLoading={isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={onCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}