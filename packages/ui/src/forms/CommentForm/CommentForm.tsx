import { Comment, CommentFor, commentTranslationValidation, commentValidation, DUMMY_ID, orDefault, Session } from "@local/shared";
import { useTheme } from "@mui/material";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { SessionContext } from "contexts/SessionContext";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { CommentFormProps } from "forms/types";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { SendIcon } from "icons";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools";
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
    const validationSchema = commentValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" });
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
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle translations
    const {
        language,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["text"],
        validationSchema: commentTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                ref={ref}
                style={{ paddingBottom: 0 }}
            >
                <TranslatedRichInput
                    disabled={isLoading}
                    language={language}
                    name="text"
                    placeholder={t("PleaseBeNice")}
                    maxChars={32768}
                    minRows={3}
                    maxRows={15}
                    actionButtons={[{
                        Icon: SendIcon,
                        onClick: () => { props.handleSubmit(); },
                    }]}
                    sxs={{
                        root: { width: "100%", background: palette.primary.dark, borderRadius: 1, overflow: "overlay", marginTop: 1 },
                        bar: { borderRadius: 0 },
                        textArea: { borderRadius: 0, paddingRight: 4, height: "100%" },
                    }}
                />
            </BaseForm>
        </>
    );
});
