import { Grid, Stack, Switch, Typography } from "@mui/material"
import { SettingsNotificationsProps } from "../types";
import { PageTitle } from "components/text";
import { getUserLanguages, PubSub, shapeProfile, usePromptBeforeUnload } from "utils";
import { mutationWrapper } from "api/utils";
import { useCustomMutation } from "api/hooks";
import { useFormik } from "formik";
import { NotificationSettings, NotificationSettingsUpdateInput } from "@shared/consts";
import { DUMMY_ID, uuid } from "@shared/uuid";
import { userValidation } from "@shared/validation";

export const SettingsNotifications = ({
    profile,
    onUpdated,
    session,
}: SettingsNotificationsProps) => {
    // Handle update
    // const [mutation] = useCustomMutation<NotificationSettings, NotificationSettingsUpdateInput>(notificationSettingsUpdate);
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
        validationSchema: userValidation.update({}),
        onSubmit: (values) => {
            // if (!profile) {
            //     PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: 'Error' });
            //     return;
            // }
            // if (!formik.isValid) {
            //     PubSub.get().publishSnack({ messageKey: 'FixErrorsBeforeSubmitting', severity: 'Error' });
            //     return;
            // }
            // const input = shapeProfile.update(profile, {
            //     id: profile.id,
            //     name: values.name,
            //     handle: selectedHandle,
            //     translations: values.translationsUpdate,
            // })
            // if (!input || Object.keys(input).length === 0) {
            //     PubSub.get().publishSnack({ messageKey: 'NoChangesMade', severity: 'Info' });
            //     return;
            // }
            // mutationWrapper<NotificationSettings, NotificationSettingsUpdateInput>({
            //     mutation,
            //     input,
            //     onError: () => { formik.setSubmitting(false) },
            // })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    return (
        <form style={{ overflow: 'hidden' }}>
            <PageTitle titleKey='Notification' titleVariables={{ count: 2 }} />
            <Grid container spacing={2}>
                <Grid item xs={12}>
                    {/* Toggle all notifications */}
                    <Stack direction="row" marginRight="auto" alignItems="center">
                        <Typography component="h2" variant="h5" textAlign="center">Toggle all</Typography>
                        {/* <Switch
                            checked={formik.values.strictlyNecessary}
                            onChange={formik.handleChange}
                            name="toggleAll"
                            sx={{
                                position: 'absolute',
                                right: '16px',
                            }}
                        /> */}
                    </Stack>
                </Grid>
            </Grid>
        </form>
    )
}