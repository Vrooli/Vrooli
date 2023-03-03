import { Checkbox, FormControlLabel, Grid } from "@mui/material";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { reminderValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { useCreateActions, usePromptBeforeUnload, useTopBar } from "utils";
import { ReminderCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, PageTitle } from "components";
import { uuid } from '@shared/uuid';
import { checkIfLoggedIn, getCurrentUser } from "utils/authentication";
import { Reminder, ReminderCreateInput } from "@shared/consts";
import { reminderCreate } from "api/generated/endpoints/reminder_create";

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

    const TopBar = useTopBar({
        display,
        session,
        titleData: {
            titleKey: 'CreateReminder',
        },
    })

    return (
        <>
            {TopBar}
            <form onSubmit={formik.handleSubmit} style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
            }}
            >
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    {/* TODO */}
                    <GridSubmitButtons
                        disabledSubmit={!isLoggedIn}
                        errors={formik.errors}
                        isCreate={true}
                        loading={formik.isSubmitting}
                        onCancel={onCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </form>
        </>
    )
}