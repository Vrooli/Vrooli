import { Reminder, ReminderCreateInput } from "@shared/consts";
import { uuid } from '@shared/uuid';
import { reminderValidation } from '@shared/validation';
import { mutationWrapper } from "api";
import { reminderCreate } from "api/generated/endpoints/reminder_create";
import { useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReminderForm } from "forms/ReminderForm.tsx/ReminderForm";
import { useContext, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeReminder } from "utils/shape/models/reminder";
import { ReminderCreateProps } from "../types";

export const ReminderCreate = ({
    display = 'page',
    zIndex = 200,
}: ReminderCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const { onCancel, onCreated } = useCreateActions<Reminder>();
    const [mutation, { loading: isLoading }] = useCustomMutation<Reminder, ReminderCreateInput>(reminderCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateReminder',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    __typename: 'Reminder' as const,
                    id: uuid(),
                }}
                onSubmit={(values, helpers) => {
                    mutationWrapper<Reminder, ReminderCreateInput>({
                        mutation,
                        input: shapeReminder.create(values),
                        onSuccess: (data) => { onCreated(data) },
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
                    onCancel={onCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}