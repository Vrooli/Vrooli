import { ChatInvite, ChatInviteCreateInput, ChatInviteUpdateInput, chatInviteValidation, endpointPostChatInvites, endpointPutChatInvites, noop } from "@local/shared";
import { Box, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ChatInviteFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useUpsertActions } from "hooks/useUpsertActions";
import { forwardRef, useState } from "react";
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

const ChatInviteForm = forwardRef<BaseFormRef | undefined, ChatInviteFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    ...props
}, ref) => {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [message, setMessage] = useState("");
    console.log("chat errors", props.errors);

    return (
        <Box ref={ref} sx={{
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
                isCreate={isCreate}
                loading={isLoading}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </Box>
    );
});


export const ChatInviteUpsert = ({
    invites,
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
}: ChatInviteUpsertProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

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
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="chat-invite-upsert-dialog"
            isOpen={isOpen}
            onClose={handleCancel}
        >
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateInvites" : "UpdateInvites")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={invites}
                onSubmit={(values, helpers) => {
                    console.log("IN SUBMIT", values);
                    if (!isCreate && invites.length === 0) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<ChatInviteCreateInput[] | ChatInviteUpdateInput[], ChatInvite[]>({
                        fetch,
                        inputs: transformChatInviteValues(values, invites, isCreate) as ChatInviteCreateInput[] | ChatInviteUpdateInput[],
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateChatInviteValues(values, invites, isCreate)}
            >
                {(formik) => <ChatInviteForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
