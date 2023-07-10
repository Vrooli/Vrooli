import { Chat, chatTranslationValidation, chatValidation, DUMMY_ID, orDefault, Session } from "@local/shared";
import { useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ChatFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ChatShape, shapeChat } from "utils/shape/models/chat";

export const chatInitialValues = (
    session: Session | undefined,
    existing?: Chat | null | undefined,
): ChatShape => ({
    __typename: "Chat" as const,
    id: DUMMY_ID,
    openToAnyoneWithInvite: false,
    organization: null,
    invites: [],
    labels: [],
    participantsDelete: [],
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "ChatTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        name: "",
        description: "",
    }]),
});

export const transformChatValues = (values: ChatShape, existing?: ChatShape) => {
    return existing === undefined
        ? shapeChat.create(values)
        : shapeChat.update(existing, values);
};

export const validateChatValues = async (values: ChatShape, existing?: ChatShape) => {
    const transformedValues = transformChatValues(values, existing);
    const validationSchema = chatValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const ChatForm = forwardRef<BaseFormRef | undefined, ChatFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle translations
    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "name"],
        validationSchema: chatTranslationValidation[isCreate ? "create" : "update"]({}),
    });


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
                    {/* TODO */}
                </FormContainer>
                <GridSubmitButtons
                    display={display}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                    zIndex={zIndex}
                />
            </BaseForm>
        </>
    );
});
