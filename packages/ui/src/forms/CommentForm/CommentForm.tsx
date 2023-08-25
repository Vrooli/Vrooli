import { Comment, CommentFor, commentTranslationValidation, commentValidation, DUMMY_ID, orDefault, Session } from "@local/shared";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { SessionContext } from "contexts/SessionContext";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { CommentFormProps } from "forms/types";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { CommentShape, shapeComment } from "utils/shape/models/comment";

export const commentInitialValues = (
    session: Session | undefined,
    objectType: CommentFor,
    objectId: string,
    language: string,
    existing?: Comment | null | undefined,
): CommentShape => ({
    __typename: "Comment" as const,
    id: DUMMY_ID,
    commentedOn: { __typename: objectType, id: objectId },
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "CommentTranslation" as const,
        id: DUMMY_ID,
        language,
        text: "",
    }]),
});

export const transformCommentValues = (values: CommentShape, existing: CommentShape, isCreate: boolean) =>
    isCreate ? shapeComment.create(values) : shapeComment.update(existing, values);

export const validateCommentValues = async (values: CommentShape, existing: CommentShape, isCreate: boolean) => {
    const transformedValues = transformCommentValues(values, existing, isCreate);
    const validationSchema = commentValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const CommentForm = forwardRef<BaseFormRef | undefined, CommentFormProps>(({
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

    // Handle translations
    const {
        language,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["text"],
        validationSchema: commentTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                ref={ref}
            >
                <FormContainer>
                    <TranslatedMarkdownInput
                        language={language}
                        name="text"
                        placeholder={t("PleaseBeNice")}
                        minRows={3}
                    />
                </FormContainer>
                <BottomActionsButtons
                    display={display}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    );
});
