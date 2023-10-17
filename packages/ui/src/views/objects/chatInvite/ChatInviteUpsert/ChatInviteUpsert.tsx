import { ChatInvite, ChatInviteCreateInput, ChatInviteStatus, ChatInviteUpdateInput, chatInviteValidation, DUMMY_ID, endpointGetChatInvite, endpointPostChatInvite, endpointPutChatInvite, Session } from "@local/shared";
import { TextField } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ChatInviteFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ChatInviteShape, shapeChatInvite } from "utils/shape/models/chatInvite";
import { ChatInviteUpsertProps } from "../types";

/** New resources must include a chat */
export type NewChatInviteShape = Partial<Omit<ChatInvite, "chat">> & {
    chat: Partial<ChatInvite["chat"]> & ({ id: string })
};

const chatInviteInitialValues = (
    session: Session | undefined,
    existing: NewChatInviteShape,
): ChatInviteShape => ({
    __typename: "ChatInvite" as const,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    id: DUMMY_ID,
    message: "",
    status: ChatInviteStatus.Pending,
    ...existing,
    user: {
        __typename: "User" as const,
        ...existing.user,
        id: existing.user?.id ?? DUMMY_ID,
    },
});

const transformChatInviteValues = (values: ChatInviteShape, existing: ChatInviteShape, isCreate: boolean) =>
    isCreate ? shapeChatInvite.create(values) : shapeChatInvite.update(existing, values);

const validateChatInviteValues = async (values: ChatInviteShape, existing: ChatInviteShape, isCreate: boolean) => {
    const transformedValues = transformChatInviteValues(values, existing, isCreate);
    const validationSchema = chatInviteValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
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
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={700}
                ref={ref}
            >
                <FormContainer>
                    <RelationshipList
                        isEditing={true}
                        objectType={"ChatInvite"}
                    />
                    <Field
                        fullWidth
                        name="message"
                        label={t("MessageOptional")}
                        as={TextField}
                    />
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors as any}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});


export const ChatInviteUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: ChatInviteUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const display = toDisplay(isOpen);
    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<ChatInvite, ChatInviteShape>({
        ...endpointGetChatInvite,
        objectType: "ChatInvite",
        overrideObject: overrideObject as ChatInvite,
        transform: (existing) => chatInviteInitialValues(session, existing as NewChatInviteShape),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<ChatInvite, ChatInviteCreateInput, ChatInviteUpdateInput>({
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
                title={t(isCreate ? "CreateInvite" : "UpdateInvite")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<ChatInviteCreateInput | ChatInviteUpdateInput, ChatInvite>({
                        fetch,
                        inputs: transformChatInviteValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateChatInviteValues(values, existing, isCreate)}
            >
                {(formik) => <ChatInviteForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
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
