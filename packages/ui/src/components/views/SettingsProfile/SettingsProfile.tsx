import { Container, Grid, TextField } from "@mui/material"
import { useMutation } from "@apollo/client";
import { user } from "graphql/generated/user";
import { useCallback, useMemo, useState } from "react";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { APP_LINKS, profileSchema as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { formatForUpdate } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { SettingsProfileProps } from "../types";
import { useLocation } from "wouter";

export const SettingsProfile = ({
    profile,
    onUpdated,
}: SettingsProfileProps) => {
    const [, setLocation] = useLocation();

    // Handle update
    const [mutation] = useMutation<user>(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            bio: '',
            username: '',
            theme: '',
            currentPassword: '',
            newPassword: '',
            newPasswordConfirmation: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForUpdate(profile, values),
                onSuccess: (response) => { onUpdated(response.data.profileUpdate) },
            })
        },
    });

    const actions: DialogActionItem[] = useMemo(() => [
        ['Save', SaveIcon, Boolean(formik.isSubmitting || !formik.isValid), true, () => { }],
        ['Cancel', CancelIcon, formik.isSubmitting, false, () => { setLocation(`${APP_LINKS.Settings}/?page=profile`, { replace: true }) }],
    ], [formik, setLocation]);
    const [formBottom, setFormBottom] = useState<number>(0);
    const handleResize = useCallback(({ height }: any) => {
        setFormBottom(height);
    }, [setFormBottom]);

    return (
        <form onSubmit={formik.handleSubmit} style={{ paddingBottom: `${formBottom}px` }}>
            <Container>
                <Grid container spacing={2} sx={{ padding: 2 }}>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            id="username"
                            name="username"
                            label="Username"
                            value={formik.values.username}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.username && Boolean(formik.errors.username)}
                            helperText={formik.touched.username && formik.errors.username}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            id="bio"
                            name="bio"
                            label="Bio"
                            multiline
                            minRows={4}
                            value={formik.values.bio}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.bio && Boolean(formik.errors.bio)}
                            helperText={formik.touched.bio && formik.errors.bio}
                        />
                    </Grid>
                </Grid>
            </Container>
            <Container>
                <Grid container spacing={2}>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            id="currentPassword"
                            name="currentPassword"
                            type="password"
                            autoComplete="password"
                            label="Current Password"
                            value={formik.values.currentPassword}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.currentPassword && Boolean(formik.errors.currentPassword)}
                            helperText={formik.touched.currentPassword && formik.errors.currentPassword}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            id="newPassword"
                            name="newPassword"
                            type="password"
                            autoComplete="new-password"
                            label="New Password"
                            value={formik.values.newPassword}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.newPassword && Boolean(formik.errors.newPassword)}
                            helperText={formik.touched.newPassword && formik.errors.newPassword}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            id="newPasswordConfirmation"
                            name="newPasswordConfirmation"
                            type="password"
                            autoComplete="new-password"
                            label="Confirm New Password"
                            value={formik.values.newPasswordConfirmation}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.newPasswordConfirmation && Boolean(formik.errors.newPasswordConfirmation)}
                            helperText={formik.touched.newPasswordConfirmation && formik.errors.newPasswordConfirmation}
                        />
                    </Grid>
                </Grid>
            </Container>
            <DialogActionsContainer actions={actions} onResize={handleResize} />
        </form>
    )
}