import { Box, IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { DUMMY_ID, DeleteType, endpointsActions, endpointsPhone, phoneValidation, updateArray, uuid, type DeleteOneInput, type Phone, type PhoneCreateInput, type SendVerificationTextInput, type Success, type ValidateVerificationTextInput } from "@vrooli/shared";
import { useFormik } from "formik";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { multiLineEllipsis } from "../../../styles.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ListContainer } from "../../containers/ListContainer.js";
import { PhoneNumberInputBase } from "../../inputs/PhoneNumberInput/PhoneNumberInput.js";
import { TextInput } from "../../inputs/TextInput/TextInput.js";
import { type PhoneListItemProps, type PhoneListProps } from "./types.js";

/**
 * Displays a list of phone numbers for the user to manage
 */
export function PhoneListItem({
    data,
    handleDelete,
    handleUpdate,
    index,
}: PhoneListItemProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const onDelete = useCallback(() => {
        handleDelete(data);
    }, [data, handleDelete]);

    // Cooldown timer
    const [cooldown, setCooldown] = useState(0);
    useEffect(() => {
        if (cooldown <= 0) return;

        const timer = setInterval(() => {
            setCooldown(cooldown - 1);
        }, 1000);

        return () => clearInterval(timer);
    }, [cooldown]);

    const [verifyMutation, { loading: loadingVerifyText }] = useLazyFetch<SendVerificationTextInput, Success>(endpointsPhone.verify);
    const sendVerificationText = useCallback(() => {
        if (cooldown > 0 || loadingVerifyText) return;

        fetchLazyWrapper<SendVerificationTextInput, Success>({
            fetch: verifyMutation,
            inputs: { phoneNumber: data.phoneNumber },
            onSuccess: () => {
                setCooldown(120);
                PubSub.get().publish("alertDialog", { messageKey: "CompleteVerificationByEnteringCode", buttons: [{ labelKey: "Ok" }] });
            },
        });
    }, [cooldown, data, loadingVerifyText, verifyMutation]);

    const [verificationCode, setVerificationCode] = useState("");
    const verificationInput = (
        <Stack direction="row" sx={{
            display: "flex",
            justifyContent: "center",
            paddingTop: 4,
        }}>
            <TextInput
                fullWidth
                name="verificationCode"
                label={t("VerificationCode")}
                placeholder={t("VerificationCodePlaceholder")}
                value={verificationCode}
                onChange={(e) => setVerificationCode(e.target.value)}
                sx={{
                    height: "56px",
                    "& .MuiInputBase-root": {
                        borderRadius: "5px 0 0 5px",
                    },
                }}
            />
            <Tooltip title={cooldown > 0 ? `${t("CanResendIn", { defaultValue: `Can send in ${cooldown}`, time: cooldown })}` : t("ResendVerificationText")}>
                <span> {/* Wrapper span to allow tooltip on disabled button */}
                    <IconButton
                        onClick={sendVerificationText}
                        disabled={cooldown > 0} // Disable button based on cooldown
                        sx={{
                            background: palette.secondary.main,
                            borderRadius: "0 5px 5px 0",
                            height: "56px",
                        }}>
                        <IconCommon
                            decorative
                            name="Refresh"
                        />
                    </IconButton>
                </span>
            </Tooltip>
        </Stack>
    );

    //TODO current twilio number only supports US and Canada (I think?). Need to add support countries list, and 
    // prevent validation process when number is not from US or Canada
    const [validateMutation, { loading: loadingValidateText }] = useLazyFetch<ValidateVerificationTextInput, Success>(endpointsPhone.validate);
    const validateText = useCallback(() => {
        if (loadingValidateText || verificationCode.length === 0) return;

        fetchLazyWrapper<ValidateVerificationTextInput, Success>({
            fetch: validateMutation,
            inputs: { phoneNumber: data.phoneNumber, verificationCode },
            onSuccess: () => {
                PubSub.get().publish("alertDialog", { messageKey: "PhoneVerifiedMaybeCreditsReceived", buttons: [{ labelKey: "Ok" }] });
                PubSub.get().publish("celebration");
                setVerificationCode("");
                handleUpdate(index, { ...data, verifiedAt: new Date() });
            },
        });
    }, [data, handleUpdate, index, loadingValidateText, validateMutation, verificationCode]);

    // If verification code is 8 characters long, automatically validate
    useEffect(() => {
        if (verificationCode.length === 8) {
            validateText();
        }
    }, [verificationCode, validateText]);

    return (
        <ListItem
            disablePadding
            sx={{
                display: "flex",
                padding: 1,
                borderBottom: `1px solid ${palette.divider}`,
            }}
        >
            <Stack direction="column" p={1} sx={{ width: "100%" }}>
                <Stack direction="row" spacing={1} alignItems="center">
                    <ListItemText
                        primary={data.phoneNumber}
                        sx={{ ...multiLineEllipsis(1) }}
                    />
                    {/* Right action buttons */}
                    <Stack direction="row" spacing={1}>
                        <Tooltip title={t("PhoneDelete")}>
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
                </Stack>
                {!data.verified && verificationInput}
                {data.verified && <Box sx={{
                    borderRadius: 1,
                    border: `2px solid ${palette.success.main}`,
                    color: palette.success.main,
                    height: "fit-content",
                    fontWeight: "bold",
                    marginTop: "auto",
                    marginBottom: "auto",
                    textAlign: "center",
                    padding: 0.25,
                    width: "fit-content",
                }}>
                    {t("Verified")}
                </Box>}
            </Stack>
        </ListItem>
    );
}

export function PhoneList({
    handleUpdate,
    numOtherVerified,
    list,
}: PhoneListProps) {
    console.log("rendering phonelist", list);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [verifyMutation, { loading: loadingVerifyText }] = useLazyFetch<SendVerificationTextInput, Success>(endpointsPhone.verify);
    const sendVerificationText = useCallback((phone: Phone) => {
        if (loadingVerifyText) return;

        fetchLazyWrapper<SendVerificationTextInput, Success>({
            fetch: verifyMutation,
            inputs: { phoneNumber: phone.phoneNumber },
            onSuccess: () => {
                PubSub.get().publish("alertDialog", { messageKey: "CompleteVerificationByEnteringCode", buttons: [{ labelKey: "Ok" }] });
            },
        });
    }, [loadingVerifyText, verifyMutation]);

    // Handle add
    const [addMutation, { loading: loadingAdd }] = useLazyFetch<PhoneCreateInput, Phone>(endpointsPhone.createOne);
    const formik = useFormik({
        initialValues: {
            id: DUMMY_ID,
            phoneNumber: "",
        },
        enableReinitialize: true,
        validationSchema: phoneValidation.create({ env: process.env.NODE_ENV as "development" | "production" }),
        onSubmit: (values) => {
            if (!formik.isValid || loadingAdd) return;
            console.log("going to submit", values, typeof values.phoneNumber, values.phoneNumber);
            fetchLazyWrapper<PhoneCreateInput, Phone>({
                fetch: addMutation,
                inputs: { ...values, id: uuid() },
                onSuccess: (data) => {
                    handleUpdate([...list, data]);
                    formik.resetForm();
                    if (!data.verified) sendVerificationText(data);
                },
                onError: () => { formik.setSubmitting(false); },
            });
        },
    });

    const [deleteMutation, { loading: loadingDelete }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    const onDelete = useCallback((phone: Phone) => {
        if (loadingDelete) return;
        // Make sure that the user has at least one other authentication method 
        if (list.length <= 1 && numOtherVerified === 0) {
            PubSub.get().publish("snack", { messageKey: "MustLeaveVerificationMethod", severity: "Error" });
            return;
        }
        // Confirmation dialog
        PubSub.get().publish("alertDialog", {
            messageKey: "PhoneDeleteConfirm",
            messageVariables: { phoneNumber: phone.phoneNumber },
            buttons: [
                {
                    labelKey: "Yes",
                    onClick: () => {
                        fetchLazyWrapper<DeleteOneInput, Success>({
                            fetch: deleteMutation,
                            inputs: { id: phone.id, objectType: DeleteType.Phone },
                            onSuccess: () => {
                                handleUpdate([...list.filter(w => w.id !== phone.id)]);
                            },
                        });
                    },
                },
                { labelKey: "Cancel" },
            ],
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete, numOtherVerified]);

    return (
        <form onSubmit={formik.handleSubmit}>
            <ListContainer
                emptyText={t("NoPhoneNumbers", { ns: "error" })}
                isEmpty={list.length === 0}
            >
                {/* Phone list */}
                {list.map((phone: Phone, index) => (
                    <PhoneListItem
                        key={`phone-${index}`}
                        data={phone}
                        index={index}
                        handleDelete={onDelete}
                        handleUpdate={(index, phone) => {
                            handleUpdate(updateArray(list, index, phone));
                        }}
                    />
                ))}
            </ListContainer>
            {/* Add new phone */}
            <Stack direction="row" sx={{
                display: "flex",
                justifyContent: "center",
                paddingTop: 4,
            }}>
                <PhoneNumberInputBase
                    fullWidth
                    id="phoneNumber"
                    name="phoneNumber"
                    label={t("NewPhoneNumber")}
                    placeholder={t("PhoneNumberPlaceholder")}
                    value={formik.values.phoneNumber}
                    onBlur={formik.handleBlur}
                    onChange={(value) => { formik.setFieldValue("phoneNumber", value); }}
                    error={formik.touched.phoneNumber && Boolean(formik.errors.phoneNumber)}
                    helperText={formik.touched.phoneNumber && formik.errors.phoneNumber}
                    setError={(error) => { formik.setFieldError("phoneNumber", error); }}
                    sx={{
                        height: "56px",
                        "& .MuiInputBase-root": {
                            borderRadius: "5px 0 0 5px",
                        },
                    }}
                />
                <IconButton
                    aria-label='add-new-phone-button'
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
