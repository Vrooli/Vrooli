import { Grid } from "@mui/material";
import { Reminder, ReminderCreateInput } from "@shared/consts";
import { uuid } from '@shared/uuid';
import { reminderValidation } from '@shared/validation';
import { reminderCreate } from "api/generated/endpoints/reminder_create";
import { useCustomMutation } from "api/hooks";
import { GridSubmitButtons, TopBar } from "components";
import { useFormik } from 'formik';
import { BaseForm } from "forms";
import { useMemo } from "react";
import { useCreateActions, usePromptBeforeUnload } from "utils";
import { checkIfLoggedIn } from "utils/authentication";
import { ReminderCreateProps } from "../types";

export const ReminderCreate = ({
    display = 'page',
    session,
    zIndex = 200,
}: ReminderCreateProps) => {
    const { onCancel, onCreated } = useCreateActions<Reminder>();

    // Handle create
    const [mutation] = useCustomMutation<Reminder, ReminderCreateInput>(reminderCreate);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
        },
        validationSchema: reminderValidation.create({}),
        onSubmit: (values) => {
            // mutationWrapper<Reminder, ReminderCreateInput>({
            //     mutation,
            //     input: shapeReminder.create({
            //         id: values.id,

            //     }),
            //     onSuccess: (data) => { onCreated(data) },
            //     onError: () => { formik.setSubmitting(false) },
            // })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'CreateReminder',
                }}
            />
            <BaseForm onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    {/* TODO */}
                    <GridSubmitButtons
                        disabledSubmit={!isLoggedIn}
                        display={display}
                        errors={formik.errors}
                        isCreate={true}
                        loading={formik.isSubmitting}
                        onCancel={onCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </BaseForm>
        </>
    )
}