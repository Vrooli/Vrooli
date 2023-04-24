import { Comment, CommentFor, Session } from "@local/consts";
import { orDefault } from "@local/utils";
import { DUMMY_ID } from "@local/uuid";
import { commentTranslationValidation, commentValidation } from "@local/validation";
import { Stack } from "@mui/material";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { TranslatedMarkdownInput } from "../../components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { CommentShape, shapeComment } from "../../utils/shape/models/comment";
import { BaseForm } from "../BaseForm/BaseForm";
import { CommentFormProps } from "../types";

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

export const transformCommentValues = (values: CommentShape, existing?: CommentShape) => {
    return existing === undefined
        ? shapeComment.create(values)
        : shapeComment.update(existing, values);
};

export const validateCommentValues = async (values: CommentShape, existing?: CommentShape) => {
    const transformedValues = transformCommentValues(values, existing);
    const validationSchema = commentValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

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
        fields: ["text"],
        validationSchema: commentTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    width: "min(700px, 100vw - 16px)",
                    margin: "auto",
                    marginBottom: "64px",
                }}
            >
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <TranslatedMarkdownInput
                        language={language}
                        name="text"
                        placeholder={t("PleaseBeNice")}
                        minRows={3}
                    />
                </Stack>
                <GridSubmitButtons
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