import { Box, Grid, Typography, useTheme } from "@mui/material";
import { Comment, CommentUpdateInput as CommentUpdateInputType } from "@shared/consts";
import { DUMMY_ID } from "@shared/uuid";
import { commentTranslationValidation, commentValidation } from '@shared/validation';
import { commentUpdate } from "api/generated/endpoints/comment_update";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { CommentDialog } from "components/dialogs/CommentDialog/CommentDialog";
import { useFormik } from "formik";
import { useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { checkIfLoggedIn } from "utils/authentication/session";
import { getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { SessionContext } from "utils/SessionContext";
import { shapeComment } from "utils/shape/models/comment";
import { MarkdownInput } from "../MarkdownInput/MarkdownInput";
import { CommentUpdateInputProps } from "../types";

/**
 * MarkdownInput/CommentContainer wrapper for creating comments
 */
export const CommentUpdateInput = ({
    comment,
    handleClose,
    language,
    objectId,
    objectType,
    onCommentUpdate,
    parent,
    zIndex,
}: CommentUpdateInputProps) => {
    const session = useContext(SessionContext);
    const { breakpoints } = useTheme();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);
    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    const [updateMutation, { loading: loadingUpdate }] = useCustomMutation<Comment, CommentUpdateInputType>(commentUpdate);
    const formik = useFormik({
        initialValues: {
            id: DUMMY_ID,
            createdFor: objectType,
            forId: objectId,
            parentId: parent?.id ?? undefined,
            translationsUpdate: [{
                id: DUMMY_ID,
                language,
                text: '',
            }],
        },
        validationSchema: commentValidation.update({}),
        onSubmit: (values) => {
            // If not logged in, open login dialog
            //TODO
            const input = shapeComment.update(comment, {
                ...comment,
                commentedOn: { __typename: values.createdFor, id: values.forId },
                translations: values.translationsUpdate,
            }, true)
            input !== undefined && mutationWrapper<Comment, CommentUpdateInputType>({
                mutation: updateMutation,
                input,
                successCondition: (data) => data !== null,
                successMessage: () => ({ key: 'CommentUpdated' }),
                onSuccess: (data) => {
                    formik.resetForm();
                    onCommentUpdate(data);
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty && formik.values.translationsUpdate.some(t => t.text.trim().length > 0) });

    const {
        onTranslationChange,
        setLanguage,
        translations,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['text'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: commentTranslationValidation.update({}),
    });
    useEffect(() => {
        setLanguage(language);
    }, [language, setLanguage]);

    // If mobile, use CommentDialog
    if (isMobile) return (
        <CommentDialog
            errorText={translations.errorText}
            handleClose={handleClose}
            handleSubmit={formik.handleSubmit}
            isAdding={false}
            isOpen={true}
            language={language}
            onTranslationChange={onTranslationChange}
            parent={parent}
            text={translations.text}
            zIndex={zIndex + 1}
        />
    )
    // Otherwise, use MarkdownInput with GridSubmitButtons
    return (
        <form>
            <Box sx={{ margin: 2 }}>
                <Typography component="h3" variant="h6" textAlign="left">{t('EditComment')}</Typography>
                <MarkdownInput
                    id="update-comment"
                    placeholder="Please be nice to each other."
                    value={translations.text}
                    minRows={3}
                    onChange={(newText: string) => onTranslationChange({ target: { name: 'text', value: newText } })}
                    error={translations.text.length > 0 && Boolean(translations.errorText)}
                    helperText={translations.text.length > 0 ? translations.errorText : ''}
                />
                <Grid container spacing={1} sx={{
                    width: 'min(100%, 400px)',
                    marginLeft: 'auto',
                    marginTop: 1,
                }}>
                    <GridSubmitButtons
                        disabledSubmit={!isLoggedIn}
                        display="dialog"
                        errors={translations.errorsWithTranslations}
                        isCreate={true}
                        loading={formik.isSubmitting || loadingUpdate}
                        onCancel={formik.resetForm}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.submitForm}
                    />
                </Grid>
            </Box>
        </form>
    )
}