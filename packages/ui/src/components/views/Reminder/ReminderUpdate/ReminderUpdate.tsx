import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useLazyQuery, useMutation } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ReminderUpdateProps } from "../types";
import { reminderValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { parseSingleItemUrl, PubSub, usePromptBeforeUnload } from "utils";
import { GridSubmitButtons, PageTitle } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { FindByIdInput, Reminder, ReminderUpdateInput } from "@shared/consts";
import { reminderFindOne, reminderUpdate } from "api/generated/endpoints/reminder";

export const ReminderUpdate = ({
    onCancel,
    onUpdated,
session,
    zIndex,
}: ReminderUpdateProps) => {
    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data, loading }] = useLazyQuery<Reminder, FindByIdInput, 'reminder'>(reminderFindOne, 'reminder');
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])
    const reminder = useMemo(() => data?.reminder, [data]);

    // Handle update
    const [mutation] = useMutation<Reminder, ReminderUpdateInput, 'reminderUpdate'>(reminderUpdate, 'reminderUpdate');
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

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12}>
                <PageTitle titleKey='UpdateReminder' session={session} />
            </Grid>
            {/* TODO */}
            <GridSubmitButtons
                errors={formik.errors}
                isCreate={false}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid>
    ), [session, formik.errors, formik.isSubmitting, formik.setSubmitting, formik.handleSubmit, onCancel]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
        }}
        >
            {loading ? (
                <Box sx={{
                    position: 'absolute',
                    top: '-5vh', // Half of toolbar height
                    width: '100%',
                    height: '100%',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}>
                    <CircularProgress size={100} color="secondary" />
                </Box>
            ) : formInput}
        </form>
    )
}