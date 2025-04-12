import { ChatInvite, ChatInviteCreateInput, ChatInviteShape, ChatInviteUpdateInput, chatInviteValidation, endpointsChatInvite, noop, noopSubmit, shapeChatInvite, validateAndGetYupErrors } from "@local/shared";
import { Box, Typography, useTheme } from "@mui/material";
import { Formik } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { RichInputBase } from "../../../components/inputs/RichInput/RichInput.js";
import { ObjectList } from "../../../components/lists/ObjectList/ObjectList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { useUpsertActions } from "../../../hooks/forms.js";
import { useHistoryState } from "../../../hooks/useHistoryState.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ChatInvitesFormProps, ChatInvitesUpsertProps } from "./types.js";

function transformChatInviteValues(values: ChatInviteShape[], existing: ChatInviteShape[], isCreate: boolean) {
    return isCreate ?
        values.map((value) => shapeChatInvite.create(value)) :
        values.map((value, index) => shapeChatInvite.update(existing[index], value)); // Assumes the dialog doesn't change the order or remove items
}

async function validateChatInviteValues(values: ChatInviteShape[], existing: ChatInviteShape[], isCreate: boolean) {
    const transformedValues = transformChatInviteValues(values, existing, isCreate);
    const validationSchema = chatInviteValidation[isCreate ? "create" : "update"]({ env: process.env.NODE_ENV });
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
    console.log("chatinvitesupsert render!", props.errors, values);
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const [message, setMessage] = useHistoryState("chat-invite-message", "");
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const { handleCancel, handleCompleted } = useUpsertActions<ChatInvite[]>({
        display,
        isCreate,
        objectType: "ChatInvite",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<ChatInvite[], ChatInviteCreateInput[], ChatInviteUpdateInput[]>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsChatInvite.createOne,
        endpointUpdate: endpointsChatInvite.updateOne,
    });
    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publish("snack", { messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!isCreate && !existing) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        setMessage("");
        if (isMutate) {
            fetchLazyWrapper<ChatInviteCreateInput[] | ChatInviteUpdateInput[], ChatInvite[]>({
                fetch,
                inputs: transformChatInviteValues(values, existing, isCreate) as ChatInviteCreateInput[] | ChatInviteUpdateInput[],
                onSuccess: (data) => { handleCompleted(data); },
                onCompleted: () => { props.setSubmitting(false); },
            });
        } else {
            handleCompleted(values as ChatInvite[]);
        }
    }, [disabled, existing, fetch, handleCompleted, isCreate, isMutate, props, setMessage, values]);

    const [inputFocused, setInputFocused] = useState(false);
    const onFocus = useCallback(() => { setInputFocused(true); }, []);
    const onBlur = useCallback(() => { setInputFocused(false); }, []);

    return (
        <MaybeLargeDialog
            display={display}
            id="chat-invite-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
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
                    <RichInputBase
                        disabled={values.length <= 0}
                        fullWidth
                        maxChars={4096}
                        maxRows={inputFocused ? 10 : 2}
                        minRows={1}
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
                            topBar: { borderRadius: 0, paddingLeft: isMobile ? "20px" : 0, paddingRight: isMobile ? "20px" : 0 },
                            bottomBar: { paddingLeft: isMobile ? "20px" : 0, paddingRight: isMobile ? "20px" : 0 },
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
                    onSubmit={onSubmit}
                />
            </Box>
        </MaybeLargeDialog>
    );
}

export function ChatInvitesUpsert({
    invites,
    isCreate,
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
                isReadLoading={false}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
