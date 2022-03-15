import { Box, Container, Grid, Stack, TextField, Typography } from "@mui/material"
import { useMutation } from "@apollo/client";
import { user } from "graphql/generated/user";
import { useCallback, useMemo, useState } from "react";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { APP_LINKS, profileUpdateSchema as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { formatForUpdate } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { SettingsNotificationsProps } from "../types";
import { containerShadow } from "styles";
import { HelpButton } from "components/buttons";

const helpText = 
`Notification preferences set the types and frequency of notifcations you receive. More customizations will be available in the near future.  
`

const TERTIARY_COLOR = '#95f3cd';

export const SettingsNotifications = ({
    profile,
    onUpdated,
}: SettingsNotificationsProps) => {

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
                <form style={{ overflow: 'hidden' }}>
                    {/* Title */}
                    <Stack direction="row" justifyContent="center" alignItems="center" sx={{
                        background: (t) => t.palette.primary.dark,
                        color: (t) => t.palette.primary.contrastText,
                        padding: 0.5,
                        marginBottom: 2,
                    }}>
                        <Typography component="h1" variant="h3">Notification</Typography>
                        <HelpButton markdown={helpText} sx={{ fill: TERTIARY_COLOR }} />
                    </Stack>
                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center'}}>
                        Coming soon
                    </Box>
                </form>
            </Box>
        </Box>
    )
}