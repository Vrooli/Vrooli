import { Box, Grid, Stack, Switch, Typography } from "@mui/material"
import { SettingsNotificationsProps } from "../types";
import { PageTitle } from "components/text";
import { SettingsFormData } from "pages";
import { PubSub, usePromptBeforeUnload } from "utils";
import { mutationWrapper } from "graphql/utils";
import { notificationSettingsUpdateMutation } from "graphql/mutation";
import { useMutation } from "@apollo/client";
import { useFormik } from "formik";

export const SettingsNotifications = ({
    profile,
    onUpdated,
    session,
}: SettingsNotificationsProps) => {

    // Handle update
    const [mutation] = useMutation(notificationSettingsUpdateMutation);
    const formik = useFormik({
        initialValues: {
            name: profile?.name ?? '',
            translationsUpdate: profile?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                bio: '',
            }],
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            if (!profile) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: SnackSeverity.Error });
                return;
            }
            if (!formik.isValid) {
                PubSub.get().publishSnack({ messageKey: 'FixErrorsBeforeSubmitting', severity: SnackSeverity.Error });
                return;
            }
            const input = shapeNotificationSettingsUpdate(profile, {
                id: profile.id,
                name: values.name,
                handle: selectedHandle,
                translations: values.translationsUpdate.map(t => ({
                    ...t,
                    id: t.id === DUMMY_ID ? uuid() : t.id,
                })),
            })
            if (!input || Object.keys(input).length === 0) {
                PubSub.get().publishSnack({ messageKey: 'NoChangesMade', severity: SnackSeverity.Info });
                return;
            }
            mutationWrapper<notificationSettingsUpdate_notificationSettingsUpdate, notificationSettingsUpdateVariables>({
                mutation,
                input,
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    return (
        <form style={{ overflow: 'hidden' }}>
            <PageTitle titleKey='Notifications' session={session} />
            <Grid container spacing={2}>
                <Grid item xs={12}>
                    {/* Toggle all notifications */}
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">Toggle all</Typography>
                        <Switch
                            checked={formik.values.strictlyNecessary}
                            onChange={formik.handleChange}
                            name="toggleAll"
                            sx={{
                                position: 'absolute',
                                right: '16px',
                            }}
                        />
                    </Stack>
                </Grid>
            </Grid>
        </form>
    )
}

export const settingsNotificationsFormData: SettingsFormData = {
    labels: ['Notifications', 'Notification Preferences', 'Alerts', 'Alert Preferences', 'Push Notifications'],
    items: [],
}