import { Chat, chatTranslationValidation, chatValidation, DUMMY_ID, orDefault, Session, uuid } from "@local/shared";
import { Checkbox, IconButton, InputAdornment, Stack, TextField, Typography, useTheme } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { SessionContext } from "contexts/SessionContext";
import { Field } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ChatFormProps } from "forms/types";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { CopyIcon } from "icons";
import { forwardRef, useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { uuidToBase36 } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ChatShape, shapeChat } from "utils/shape/models/chat";

export const chatInitialValues = (
    session: Session | undefined,
    existing?: Partial<Chat> | null | undefined,
): ChatShape => ({
    __typename: "Chat" as const,
    id: uuid(),
    openToAnyoneWithInvite: false,
    organization: null,
    invites: [],
    labels: [],
    messages: [],
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

export const transformChatValues = (values: ChatShape, existing: ChatShape, isCreate: boolean) =>
    isCreate ? shapeChat.create(values) : shapeChat.update(existing, values);

export const validateChatValues = async (values: ChatShape, existing: ChatShape, isCreate: boolean) => {
    const transformedValues = transformChatValues(values, existing, isCreate);
    const validationSchema = chatValidation[isCreate ? "create" : "update"]({});
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

    const url = useMemo(() => `${window.location.origin}/chat/${uuidToBase36(values.id)}`, [values.id]);
    const copyLink = useCallback(() => {
        navigator.clipboard.writeText(url);
        PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
    }, [url]);

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
                    {/* TODO relationshiplist should be for connecting organization and inviting users. Can invite non-members even if organization specified, but the search should show members first */}
                    <RelationshipList
                        isEditing={true}
                        objectType={"Chat"}
                        sx={{ marginBottom: 4 }}
                    />
                    <FormSection sx={{
                        overflowX: "hidden",
                    }}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                        />
                        <Field
                            fullWidth
                            name="name"
                            label={t("Name")}
                            as={TextField}
                        />
                        <TranslatedMarkdownInput
                            language={language}
                            maxChars={2048}
                            minRows={4}
                            name="description"
                            placeholder={t("Description")}
                        />
                    </FormSection>
                    {/* Invite link */}
                    <Stack direction="column" spacing={1}>
                        <Stack direction="row" sx={{ alignItems: "center" }}>
                            <Typography variant="h6">{t(`OpenToAnyoneWithLink${values.openToAnyoneWithInvite ? "True" : "False"}`)}</Typography>
                            <HelpButton markdown={t("OpenToAnyoneWithLinkDescription")} />
                        </Stack>
                        <Stack direction="row" spacing={0}>
                            {/* Enable/disable */}
                            {/* <Checkbox
                                id="open-to-anyone"
                                size="small"
                                name='openToAnyoneWithInvite'
                                color='secondary'

                            /> */}
                            <Field
                                name="openToAnyoneWithInvite"
                                type="checkbox"
                                as={Checkbox}
                                sx={{
                                    "&.MuiCheckbox-root": {
                                        color: palette.secondary.main,
                                    },
                                }}
                            />
                            {/* Show link with copy adornment*/}
                            <TextField
                                disabled
                                fullWidth
                                id="invite-link"
                                label={t("InviteLink")}
                                variant="outlined"
                                value={url}
                                InputProps={{
                                    endAdornment: (
                                        <InputAdornment position="end">
                                            <IconButton
                                                aria-label={t("Copy")}
                                                onClick={copyLink}
                                                size="small"
                                            >
                                                <CopyIcon />
                                            </IconButton>
                                        </InputAdornment>
                                    ),
                                }}
                            />
                        </Stack>
                    </Stack>
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});
