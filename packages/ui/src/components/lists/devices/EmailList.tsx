import { Box, IconButton, InputAdornment, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { DUMMY_ID, DeleteType, emailValidation, endpointsActions, endpointsEmail, uuid, type DeleteOneInput, type Email, type EmailCreateInput, type SendVerificationEmailInput, type Success } from "@vrooli/shared";
import { useFormik } from "formik";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { multiLineEllipsis } from "../../../styles.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ListContainer } from "../../containers/ListContainer.js";
import { TextInput } from "../../inputs/TextInput/TextInput.js";
import { type EmailListItemProps, type EmailListProps } from "./types.js";

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
    const { t } = useTranslation();

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
            <Stack direction="column" spacing={1} pl={2} mr="auto">
                <ListItemText
                    primary={data.emailAddress}
                    sx={{ ...multiLineEllipsis(1) }}
                />
                {/* Verified indicator */}
                <Box sx={{
                    borderRadius: 1,
                    border: `2px solid ${data.verified ? palette.success.main : palette.error.main}`,
                    color: data.verified ? palette.success.main : palette.error.main,
                    height: "fit-content",
                    fontWeight: "bold",
                    marginTop: "auto",
                    marginBottom: "auto",
                    textAlign: "center",
                    padding: 0.25,
                    width: "fit-content",
                }}>
                    {t(data.verified ? "Verified" : "VerifiedNot")}
                </Box>
            </Stack>
            {/* Right action buttons */}
            <Stack direction="row" spacing={1}>
                {!data.verified && <Tooltip title={t("ResendVerificationEmail")}>
                    <IconButton
                        onClick={onVerify}
                    >
                        <IconCommon
                            decorative
                            fill="error.main"
                            name="Complete"
                        />
                    </IconButton>
                </Tooltip>}
                <Tooltip title={t("EmailDelete")}>
                    <IconButton
                        onClick={onDelete}
                    >
                        <IconCommon
                            decorative
                            fill="secondary.main"
                            name="Delete"
                        />
                    </IconButton>
                </Tooltip>
            </Stack>
        </ListItem>
    );
}

const emailInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                decorative
                name="Email"
            />
        </InputAdornment>
    ),
} as const;

export function EmailList({
    handleUpdate,
    numOtherVerified,
    list,
}: EmailListProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle add
    const [addMutation, { loading: loadingAdd }] = useLazyFetch<EmailCreateInput, Email>(endpointsEmail.createOne);
    const formik = useFormik({
        initialValues: {
            id: DUMMY_ID,
            emailAddress: "",
        },
        enableReinitialize: false,
        validationSchema: emailValidation.create({ env: process.env.NODE_ENV as "development" | "production" }),
        onSubmit: (values) => {
            if (!formik.isValid || loadingAdd) return;
            fetchLazyWrapper<EmailCreateInput, Email>({
                fetch: addMutation,
                inputs: { ...values, id: uuid() },
                onSuccess: (data) => {
                    PubSub.get().publish("snack", { messageKey: "CompleteVerificationInEmail", severity: "Info" });
                    handleUpdate([...list, data]);
                    formik.resetForm();
                },
                onError: () => { formik.setSubmitting(false); },
            });
        },
    });

    const [deleteMutation, { loading: loadingDelete }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    const onDelete = useCallback((email: Email) => {
        if (loadingDelete) return;
        // Make sure that the user has at least one other authentication method 
        if (list.length <= 1 && numOtherVerified === 0) {
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
    }, [deleteMutation, handleUpdate, list, loadingDelete, numOtherVerified]);

    const [verifyMutation, { loading: loadingVerifyEmail }] = useLazyFetch<SendVerificationEmailInput, Success>(endpointsEmail.verify);
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
                    InputProps={emailInputProps}
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
                    <IconCommon
                        decorative
                        name="Add"
                    />
                </IconButton>
            </Stack>
        </form>
    );
}
