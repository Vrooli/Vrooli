import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ReminderUpdateProps } from "../types";
import { reminderValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { parseSingleItemUrl, PubSub, usePromptBeforeUnload, useUpdateActions } from "utils";
import { GridSubmitButtons, Header, TopBar } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { FindByIdInput, Reminder, ReminderUpdateInput } from "@shared/consts";
import { reminderFindOne } from "api/generated/endpoints/reminder_findOne";
import { reminderUpdate } from "api/generated/endpoints/reminder_update";
import { BaseForm } from "forms";

export const ReminderUpdate = ({
    display = 'page',
    session,
    zIndex = 200,
}: ReminderUpdateProps) => {
    const { onCancel, onUpdated } = useUpdateActions<Reminder>();

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: reminder, loading }] = useCustomLazyQuery<Reminder, FindByIdInput>(reminderFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    // Handle update
    const [mutation] = useCustomMutation<Reminder, ReminderUpdateInput>(reminderUpdate);
    const formik = useFormik({
        initialValues: {
            id: reminder?.id ?? uuid(),
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: reminderValidation.update({}),
        onSubmit: (values) => {
            if (!reminder) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadReminder', severity: 'Error' });
                return;
            }
            // mutationWrapper<Reminder, ReminderUpdateInput>({
            //     mutation,
            //     input: shapeReminder.update(reminder, {
            //         id: reminder.id,
            //     }),
            //     onSuccess: (data) => { onUpdated(data) },
            //     onError: () => { formik.setSubmitting(false) },
            // })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                session={session}
                titleData={{
                    titleKey: 'UpdateReminder',
                }}
            />
            <BaseForm isLoading={loading} onSubmit={formik.handleSubmit}>
                <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                    {/* TODO */}
                    <GridSubmitButtons
                        display={display}
                        errors={formik.errors}
                        isCreate={false}
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