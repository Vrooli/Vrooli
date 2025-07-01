import { useTheme } from "@mui/material";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { chatInviteFormConfig, noop, noopSubmit, validateAndGetYupErrors, type ChatInvite, type ChatInviteShape } from "@vrooli/shared";
import { Formik } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { Dialog } from "../../../components/dialogs/Dialog/Dialog.js";
import { AdvancedInputBase } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { ObjectList } from "../../../components/lists/ObjectList/ObjectList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { useHistoryState } from "../../../hooks/useHistoryState.js";
import { useIsMobile } from "../../../hooks/useIsMobile.js";
import { useStandardBatchUpsertForm } from "../../../hooks/useStandardBatchUpsertForm.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { type ChatInvitesFormProps, type ChatInvitesUpsertProps } from "./types.js";

export function transformChatInviteValues(values: ChatInviteShape[], existing: ChatInviteShape[], isCreate: boolean) {
    return isCreate ?
        values.map((value) => chatInviteFormConfig.transformations.shapeToInput.create?.(value)) :
        values.map((value, index) => chatInviteFormConfig.transformations.shapeToInput.update?.(existing[index], value)); // Assumes the dialog doesn't change the order or remove items
}

async function validateChatInviteValues(values: ChatInviteShape[], existing: ChatInviteShape[], isCreate: boolean) {
    const transformedValues = transformChatInviteValues(values, existing, isCreate);
    const validationSchema = chatInviteFormConfig.validation.schema[isCreate ? "create" : "update"]({ env: process.env.NODE_ENV });
    const result = await Promise.all(transformedValues.map(async (value) => await validateAndGetYupErrors(validationSchema, value)));

    // Filter and combine the result into one object with only error results
    const combinedResult = result.reduce((acc, curr, index) => {
        if (Object.keys(curr).length > 0) {  // check if the object has any keys (errors)
            acc[index] = curr;
        }
        return acc;
    }, {} as any);

    return combinedResult;
}

function ChatInvitesForm({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isMutate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: ChatInvitesFormProps) {
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const [message, setMessage] = useHistoryState("chat-invite-message", "");
    const isMobile = useIsMobile();
    const isMobileOld = useWindowSize(({ width }) => width <= breakpoints.values.md);

    // Use the standardized batch form hook
    const {
        isLoading,
        handleCancel,
        handleCompleted,
        onSubmit,
    } = useStandardBatchUpsertForm({
        objectType: "ChatInvite",
        transformFunction: transformChatInviteValues,
        validateFunction: validateChatInviteValues,
        endpoints: {
            create: chatInviteFormConfig.endpoints.createOne,
            update: chatInviteFormConfig.endpoints.updateOne,
        },
    }, {
        values,
        existing,
        isCreate,
        display,
        disabled,
        isMutate,
        isReadLoading,
        isSubmitting: props.isSubmitting,
        handleUpdate,
        setSubmitting: props.setSubmitting,
        onCancel,
        onCompleted,
        onDeleted,
        onClose,
    });

    const handleSubmit = useCallback(() => {
        setMessage("");
        onSubmit();
    }, [setMessage, onSubmit]);

    const [inputFocused, setInputFocused] = useState(false);
    const onFocus = useCallback(() => { setInputFocused(true); }, []);
    const onBlur = useCallback(() => { setInputFocused(false); }, []);

    const dialogContent = (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateInvites" : "UpdateInvites")}
            />
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                height: "100%",
                overflow: "hidden",
            }}>
                <Typography variant="h6" p={2}>{t("InvitesGoingTo")}</Typography>
                <Box sx={{
                    display: "flex",
                    flexDirection: "column",
                    flexGrow: 1,
                    margin: "auto",
                    overflowY: "auto",
                    width: "min(500px, 100vw)",
                    pointerEvents: "none",
                }}>
                    <ObjectList
                        loading={false}
                        items={values}
                        keyPrefix="invite-list-item"
                        onAction={noop}
                        onClick={noop}
                    />
                </Box>
                <Box mt={4}>
                    <Typography variant="h6" p={2}>{t("Message", { count: 1 })}</Typography>
                    <AdvancedInputBase
                        disabled={values.length <= 0}
                        features={{ maxChars: 4096, maxRowsCollapsed: 2, maxRowsExpanded: 10, minRowsCollapsed: 1 }}
                        onBlur={onBlur}
                        onChange={setMessage}
                        onFocus={onFocus}
                        name="message"
                        sxs={{
                            root: {
                                background: palette.primary.main,
                                color: palette.primary.contrastText,
                                paddingBottom: 2,
                                maxHeight: "min(75vh, 500px)",
                                width: "-webkit-fill-available",
                                margin: "0",
                            },
                            topBar: { borderRadius: 0, paddingLeft: isMobileOld ? "20px" : 0, paddingRight: isMobileOld ? "20px" : 0 },
                            bottomBar: { paddingLeft: isMobileOld ? "20px" : 0, paddingRight: isMobileOld ? "20px" : 0 },
                            inputRoot: {
                                border: "none",
                                background: palette.background.paper,
                            },
                        }}
                        value={message}
                    />
                </Box>
                <BottomActionsButtons
                    display={display}
                    errors={props.errors as any}
                    hideButtons={disabled}
                    isCreate={isCreate}
                    loading={isLoading}
                    onCancel={handleCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={handleSubmit}
                />
            </Box>
        </>
    );

    return (
        <Dialog
            isOpen={isOpen}
            onClose={onClose}
            size={isMobile ? "full" : "lg"}
        >
            {dialogContent}
        </Dialog>
    );
}

export function ChatInvitesUpsert({
    invites,
    isCreate,
    isMutate,
    isOpen,
    ...props
}: ChatInvitesUpsertProps) {

    async function validateValues(values: ChatInviteShape[]) {
        return await validateChatInviteValues(values, invites, isCreate);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={invites}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <ChatInvitesForm
                disabled={false}
                existing={invites}
                handleUpdate={() => { }}
                isCreate={isCreate}
                isMutate={isMutate}
                isReadLoading={false}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
