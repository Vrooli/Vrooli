import { ChatInvite, ChatInviteCreateInput, ChatInviteUpdateInput, chatInviteValidation, endpointPostChatInvites, endpointPutChatInvites, noop, noopSubmit } from "@local/shared";
import { Box, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { ChatInviteFormProps } from "forms/types";
import { useConfirmBeforeLeave } from "hooks/useConfirmBeforeLeave";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ChatInviteShape, shapeChatInvite } from "utils/shape/models/chatInvite";
import { ChatInviteUpsertProps } from "../types";

const transformChatInviteValues = (values: ChatInviteShape[], existing: ChatInviteShape[], isCreate: boolean) =>
    isCreate ?
        values.map((value) => shapeChatInvite.create(value)) :
        values.map((value, index) => shapeChatInvite.update(existing[index], value)); // Assumes the dialog doesn't change the order or remove items

const validateChatInviteValues = async (values: ChatInviteShape[], existing: ChatInviteShape[], isCreate: boolean) => {
    const transformedValues = transformChatInviteValues(values, existing, isCreate);
    const validationSchema = chatInviteValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await Promise.all(transformedValues.map(async (value) => await validateAndGetYupErrors(validationSchema, value)));

    // Filter and combine the result into one object with only error results
    const combinedResult = result.reduce((acc, curr, index) => {
        if (Object.keys(curr).length > 0) {  // check if the object has any keys (errors)
            acc[index] = curr;
        }
        return acc;
    }, {} as any);

    return combinedResult;
};

const ChatInviteForm = ({
    disabled,
    dirty,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onCompleted,
    onDeleted,
    values,
    ...props
}: ChatInviteFormProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);
    const { palette } = useTheme();
    const [message, setMessage] = useState("");

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<ChatInvite[], ChatInviteCreateInput[], ChatInviteUpdateInput[]>({
        display,
        endpointCreate: endpointPostChatInvites,
        endpointUpdate: endpointPutChatInvites,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { handleClose } = useConfirmBeforeLeave({ handleCancel, shouldPrompt: dirty });
    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!isCreate && existing.length === 0) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        fetchLazyWrapper<ChatInviteCreateInput[] | ChatInviteUpdateInput[], ChatInvite[]>({
            fetch,
            inputs: transformChatInviteValues(values, existing, isCreate) as ChatInviteCreateInput[] | ChatInviteUpdateInput[],
            onSuccess: (data) => { handleCompleted(data); },
            onCompleted: () => { props.setSubmitting(false); },
        });
    }, [disabled, existing, fetch, handleCompleted, isCreate, props, values]);

    return (
        <MaybeLargeDialog
            display={display}
            id="chat-invite-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateInvites" : "UpdateInvites")}
            />
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                height: "100%",
                overflow: "hidden",
            }}>
                <Typography variant="h5" p={2}>{t("InvitesGoingTo")}</Typography>
                <Box sx={{
                    display: "flex",
                    flexDirection: "column",
                    flexGrow: 1,
                    margin: "auto",
                    overflowY: "auto",
                    width: "min(500px, 100vw)",
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
                    <Typography variant="h6" p={2}>{t("MessageOptional")}</Typography>
                    <RichInputBase
                        disabled={values.length <= 0}
                        fullWidth
                        maxChars={4096}
                        minRows={1}
                        onChange={setMessage}
                        name="message"
                        sxs={{
                            root: {
                                background: palette.primary.main,
                                color: palette.primary.contrastText,
                                paddingBottom: 2,
                                maxHeight: "min(50vh, 500px)",
                                width: "-webkit-fill-available",
                                margin: "0",
                            },
                            bar: { borderRadius: 0 },
                            textArea: { paddingRight: 4, border: "none" },
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
};

export const ChatInviteUpsert = ({
    invites,
    isCreate,
    isOpen,
    ...props
}: ChatInviteUpsertProps) => {

    return (
        <Formik
            enableReinitialize={true}
            initialValues={invites}
            onSubmit={noopSubmit}
            validate={async (values) => await validateChatInviteValues(values, invites, isCreate)}
        >
            {(formik) => <ChatInviteForm
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
};
