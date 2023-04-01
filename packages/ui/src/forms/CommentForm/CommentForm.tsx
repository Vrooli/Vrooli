import { Stack } from "@mui/material";
import { Comment, CommentFor, Session } from "@shared/consts";
import { DUMMY_ID } from "@shared/uuid";
import { commentTranslationValidation, commentValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { CommentFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { CommentShape, shapeComment } from "utils/shape/models/comment";

export const commentInitialValues = (
    session: Session | undefined,
    objectType: CommentFor,
    objectId: string,
    language: string,
    existing?: Comment | null | undefined
): CommentShape => ({
    __typename: 'Comment' as const,
    id: DUMMY_ID,
    commentedOn: { __typename: objectType, id: objectId },
    translations: [{
        id: DUMMY_ID,
        language,
        text: '',
    }],
    ...existing,
});

export const transformCommentValues = (o: CommentShape, u?: CommentShape) => {
    return u === undefined
        ? shapeComment.create(o)
        : shapeComment.update(o, u)
}

export const validateCommentValues = async (values: CommentShape, isCreate: boolean) => {
    const transformedValues = transformCommentValues(values);
    const validationSchema = isCreate
        ? commentValidation.create({})
        : commentValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

export const CommentForm = forwardRef<any, CommentFormProps>(({
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
    const { t } = useTranslation();

    // Handle translations
    const {
        language,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['text'],
        validationSchema: isCreate ? commentTranslationValidation.create({}) : commentTranslationValidation.update({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    maxWidth: '700px',
                    marginBottom: '64px',
                }}
            >
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <TranslatedMarkdownInput
                        language={language}
                        name="text"
                        placeholder={t(`PleaseBeNice`)}
                        minRows={3}
                    />
                </Stack>
                <GridSubmitButtons
                    display={display}
                    errors={{
                        ...props.errors,
                        ...translationErrors,
                    }}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    )
})