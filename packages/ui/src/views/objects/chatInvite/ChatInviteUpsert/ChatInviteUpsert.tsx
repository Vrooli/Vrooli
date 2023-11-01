import { ChatInvite, ChatInviteCreateInput, ChatInviteUpdateInput, chatInviteValidation, endpointPostChatInvites, endpointPutChatInvites, noop } from "@local/shared";
import { Box, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ChatInviteFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useUpsertActions } from "hooks/useUpsertActions";
import { forwardRef, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ChatInviteShape, shapeChatInvite } from "utils/shape/models/chatInvite";
import { ChatInviteUpsertProps } from "../types";

/** New resources must include a chat */
export type NewChatInviteShape = Partial<Omit<ChatInvite, "chat">> & {
    __typename: "ChatInvite";
    chat: Partial<ChatInvite["chat"]> & ({ id: string })
};

const chatInviteInitialValues = (
    session: Session | undefined,
    existing: NewChatInviteShape,
): ChatInviteShape => ({
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    id: DUMMY_ID,
    message: "",
    status: ChatInviteStatus.Pending,
    ...existing,
    chat: {
        __typename: "Chat" as const,
        ...existing.chat,
    },
    user: {
        __typename: "User" as const,
        ...existing.user,
        id: existing.user?.id ?? DUMMY_ID,
    },
});

const transformChatInviteValues = (values: ChatInviteShape[], existing: ChatInviteShape[], isCreate: boolean) =>
    isCreate ?
        values.map((value) => shapeChatInvite.create(value)) :
        values.map((value, index) => shapeChatInvite.update(existing[index], value)); // Assumes the dialog doesn't change the order or remove items

const validateChatInviteValues = async (values: ChatInviteShape[], existing: ChatInviteShape[], isCreate: boolean) => {
    const transformedValues = transformChatInviteValues(values, existing, isCreate);
    const validationSchema = chatInviteValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await Promise.all(transformedValues.map(async (value) => await validateAndGetYupErrors(validationSchema, value)));
    return result; //TODO probably need to combine into object
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
    const [message, setMessage] = useState("");


    return (
        <>
            <ObjectList
                loading={false}
                items={values}
                keyPrefix="invite-list-item"
                onAction={noop}
                onClick={noop}
            />
            <RichInputBase
                disabled={values.length <= 0}
                fullWidth
                maxChars={4096}
                minRows={1}
                onChange={setMessage}
                name="message"
                sxs={{
                    root: {
                        maxHeight: "min(50vh, 500px)",
                        width: "min(700px, 100%)",
                        margin: "auto",
                    },
                    bar: { borderRadius: 0 },
                    textArea: { paddingRight: 4, border: "none" },
                }}
                value={message}
            />
            <BottomActionsButtons
                display={display}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
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
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);
    const { palette } = useTheme();


    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<ChatInvite[], ChatInviteCreateInput[], ChatInviteUpdateInput[]>({
        display,
        endpointCreate: endpointPostChatInvite,
        endpointUpdate: endpointPutChatInvite,
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
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateInvites" : "UpdateInvites")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={invites}
                onSubmit={(values, helpers) => {
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
