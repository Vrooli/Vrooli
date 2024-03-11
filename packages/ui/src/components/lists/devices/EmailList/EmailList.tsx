import { DeleteOneInput, DeleteType, Email, EmailCreateInput, emailValidation, endpointPostDeleteOne, endpointPostEmail, endpointPostEmailVerification, SendVerificationEmailInput, Success } from "@local/shared";
import { Box, IconButton, InputAdornment, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { useFormik } from "formik";
import { useLazyFetch } from "hooks/useLazyFetch";
import { AddIcon, CompleteIcon, DeleteIcon, EmailIcon } from "icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis } from "styles";
import { PubSub } from "utils/pubsub";
import { EmailListItemProps, EmailListProps } from "../types";

const Status = {
    NotVerified: "#a71c2d", // Red
    Verified: "#19972b", // Green
};

/**
 * Displays a list of emails for the user to manage
 */
export function EmailListItem({
    handleDelete,
    handleVerify,
    index,
    data,
}: EmailListItemProps) {
    const { palette } = useTheme();

    const onDelete = useCallback(() => {
        handleDelete(data);
    }, [data, handleDelete]);

    const onVerify = useCallback(() => {
        handleVerify(data);
    }, [data, handleVerify]);

    return (
        <ListItem
            disablePadding
            sx={{
                display: "flex",
                padding: 1,
                borderBottom: `1px solid ${palette.divider}`,
            }}
        >
            {/* Left informational column */}
            <Stack direction="column" spacing={1} pl={2} sx={{ marginRight: "auto" }}>
                <ListItemText
                    primary={data.emailAddress}
                    sx={{ ...multiLineEllipsis(1) }}
                />
                {/* Verified indicator */}
                <Box sx={{
                    borderRadius: 1,
                    border: `2px solid ${data.verified ? Status.Verified : Status.NotVerified}`,
                    color: data.verified ? Status.Verified : Status.NotVerified,
                    height: "fit-content",
                    fontWeight: "bold",
                    marginTop: "auto",
                    marginBottom: "auto",
                    textAlign: "center",
                    padding: 0.25,
                    width: "fit-content",
                }}>
                    {data.verified ? "Verified" : "Not Verified"}
                </Box>
            </Stack>
            {/* Right action buttons */}
            <Stack direction="row" spacing={1}>
                {!data.verified && <Tooltip title="Resend email verification">
                    <IconButton
                        onClick={onVerify}
                    >
                        <CompleteIcon fill={Status.NotVerified} />
                    </IconButton>
                </Tooltip>}
                <Tooltip title="Delete Email">
                    <IconButton
                        onClick={onDelete}
                    >
                        <DeleteIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
            </Stack>
        </ListItem>
    );
}

export const EmailList = ({
    handleUpdate,
    numVerifiedWallets,
    list,
}: EmailListProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle add
    const [addMutation, { loading: loadingAdd }] = useLazyFetch<EmailCreateInput, Email>(endpointPostEmail);
    const formik = useFormik({
        initialValues: {
            emailAddress: "",
        },
        enableReinitialize: true,
        validationSchema: emailValidation.create({ env: process.env.NODE_ENV as "development" | "production" }),
        onSubmit: (values) => {
            if (!formik.isValid || loadingAdd) return;
            fetchLazyWrapper<EmailCreateInput, Email>({
                fetch: addMutation,
                inputs: {
                    emailAddress: values.emailAddress,
                },
                onSuccess: (data) => {
                    PubSub.get().publish("snack", { messageKey: "CompleteVerificationInEmail", severity: "Info" });
                    handleUpdate([...list, data]);
                    formik.resetForm();
                },
                onError: () => { formik.setSubmitting(false); },
            });
        },
    });

    const [deleteMutation, { loading: loadingDelete }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const onDelete = useCallback((email: Email) => {
        if (loadingDelete) return;
        // Make sure that the user has at least one other authentication method 
        // (i.e. one other email or one other wallet)
        if (list.length <= 1 && numVerifiedWallets === 0) {
            PubSub.get().publish("snack", { messageKey: "MustLeaveVerificationMethod", severity: "Error" });
            return;
        }
        // Confirmation dialog
        PubSub.get().publish("alertDialog", {
            messageKey: "EmailDeleteConfirm",
            messageVariables: { emailAddress: email.emailAddress },
            buttons: [
                {
                    labelKey: "Yes",
                    onClick: () => {
                        fetchLazyWrapper<DeleteOneInput, Success>({
                            fetch: deleteMutation,
                            inputs: { id: email.id, objectType: DeleteType.Email },
                            onSuccess: () => {
                                handleUpdate([...list.filter(w => w.id !== email.id)]);
                            },
                        });
                    },
                },
                { labelKey: "Cancel" },
            ],
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete, numVerifiedWallets]);

    const [verifyMutation, { loading: loadingVerifyEmail }] = useLazyFetch<SendVerificationEmailInput, Success>(endpointPostEmailVerification);
    const sendVerificationEmail = useCallback((email: Email) => {
        if (loadingVerifyEmail) return;
        fetchLazyWrapper<SendVerificationEmailInput, Success>({
            fetch: verifyMutation,
            inputs: { emailAddress: email.emailAddress },
            onSuccess: () => {
                PubSub.get().publish("snack", { messageKey: "CompleteVerificationInEmail", severity: "Info" });
            },
        });
    }, [loadingVerifyEmail, verifyMutation]);

    return (
        <form onSubmit={formik.handleSubmit}>
            <ListContainer
                emptyText={t("NoEmails", { ns: "error" })}
                isEmpty={list.length === 0}
                sx={{ maxWidth: "500px" }}
            >
                {/* Email list */}
                {list.map((email: Email, index) => (
                    <EmailListItem
                        key={`email-${index}`}
                        data={email}
                        index={index}
                        handleDelete={onDelete}
                        handleVerify={sendVerificationEmail}
                    />
                ))}
            </ListContainer>
            {/* Add new email */}
            <Stack direction="row" sx={{
                display: "flex",
                justifyContent: "center",
                paddingTop: 4,
            }}>
                <TextInput
                    autoComplete='email'
                    fullWidth
                    id="emailAddress"
                    name="emailAddress"
                    label={t("NewEmailAddress")}
                    placeholder={t("EmailPlaceholder")}
                    value={formik.values.emailAddress}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.emailAddress && Boolean(formik.errors.emailAddress)}
                    helperText={formik.touched.emailAddress && formik.errors.emailAddress}
                    InputProps={{
                        startAdornment: (
                            <InputAdornment position="start">
                                <EmailIcon />
                            </InputAdornment>
                        ),
                    }}
                    sx={{
                        height: "56px",
                        "& .MuiInputBase-root": {
                            borderRadius: "5px 0 0 5px",
                        },
                    }}
                />
                <IconButton
                    aria-label='add-new-email-button'
                    type='submit'
                    sx={{
                        background: palette.secondary.main,
                        borderRadius: "0 5px 5px 0",
                        height: "56px",
                    }}>
                    <AddIcon />
                </IconButton>
            </Stack>
        </form>
    );
};
