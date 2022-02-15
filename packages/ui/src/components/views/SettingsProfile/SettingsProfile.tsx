import { Box, Container, Grid, TextField, Typography } from "@mui/material"
import { useMutation } from "@apollo/client";
import { user } from "graphql/generated/user";
import { useMemo } from "react";
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
import { containerShadow } from "styles";

export const SettingsProfile = ({
    profile,
    onUpdated,
}: SettingsProfileProps) => {
    const [, setLocation] = useLocation();

    // Handle update
    const [mutation] = useMutation<user>(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            bio: profile?.bio ?? "",
            username: profile?.username ?? "",
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            if (!formik.isValid) return;
            mutationWrapper({
                mutation,
                input: formatForUpdate(profile, { ...values }),
                onSuccess: (response) => { onUpdated(response.data.profileUpdate) },
            })
        },
    });

    const actions: DialogActionItem[] = useMemo(() => [
        ['Save', SaveIcon, !formik.touched || formik.isSubmitting, true, () => { }],
        ['Cancel', CancelIcon, !formik.touched || formik.isSubmitting, false, () => { setLocation(`${APP_LINKS.Settings}/?page=profile`, { replace: true }) }],
    ], [formik, setLocation]);

    return (
        <Box sx={{ minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
            <Box sx={{
                ...containerShadow,
                borderRadius: 2,
                overflow: 'overlay',
                marginTop: '-5vh',
                background: (t) => t.palette.background.default,
                width: 'min(100%, 700px)',
            }}>
                <form onSubmit={formik.handleSubmit} style={{ overflow: 'hidden' }}>
                    {/* Title */}
                    <Box sx={{
                        background: (t) => t.palette.primary.dark,
                        color: (t) => t.palette.primary.contrastText,
                        padding: 0.5,
                        marginBottom: 2,
                    }}>
                        <Typography component="h1" variant="h3" textAlign="center">Update Profile</Typography>
                    </Box>
                    <Container sx={{ paddingBottom: 2 }}>
                        <Grid container spacing={2}>
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
                    <DialogActionsContainer fixed={false} actions={actions} />
                </form>
            </Box>
        </Box>
    )
}