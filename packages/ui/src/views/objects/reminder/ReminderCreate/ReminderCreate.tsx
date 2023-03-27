import { Reminder, ReminderCreateInput } from "@shared/consts";
import { reminderValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { reminderCreate } from "api/generated/endpoints/reminder_create";
import { useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderForm } from "forms/ReminderForm.tsx/ReminderForm";
import { useContext, useMemo, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeReminder } from "utils/shape/models/reminder";
import { reminderInitialValues } from "..";
import { ReminderCreateProps } from "../types";

export const ReminderCreate = ({
    display = 'page',
    index,
    onCancel,
    onCreated,
    reminderListId,
    zIndex = 200,
}: ReminderCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => reminderInitialValues(session, reminderListId), [reminderListId, session]);
    const { handleCancel, handleCreated } = useCreateActions<Reminder>(display, onCancel, onCreated);
    const [mutation, { loading: isLoading }] = useCustomMutation<Reminder, ReminderCreateInput>(reminderCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'CreateReminder',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    mutationWrapper<Reminder, ReminderCreateInput>({
                        mutation,
                        input: shapeReminder.create(values),
                        onSuccess: (data) => { handleCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={reminderValidation.create({})}
            >
                {(formik) => <ReminderForm
                    display={display}
                    isCreate={true}
                    isLoading={isLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}