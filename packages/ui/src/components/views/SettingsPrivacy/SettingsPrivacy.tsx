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
import { SettingsPrivacyProps } from "../types";

export const SettingsPrivacy = ({
    profile,
    onUpdated,
}: SettingsPrivacyProps) => {

    return (
        <form>
            TODO
        </form>
    )
}