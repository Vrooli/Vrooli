import { useMutation } from "graphql/hooks";
import { DUMMY_ID } from "@shared/uuid";
import { CommentDialog } from "components/dialogs"
import { useCallback, useMemo } from "react";
import { handleTranslationChange, shapeComment, usePromptBeforeUnload, useTranslatedFields, useWindowSize } from "utils";
import { CommentUpdateInputProps } from "../types"
import { commentValidation, commentTranslationValidation } from '@shared/validation';
import { getCurrentUser } from "utils/authentication";
import { mutationWrapper } from "graphql/utils";
import { useFormik } from "formik";
import { Box, Grid, Typography, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons";
import { MarkdownInput } from "../MarkdownInput/MarkdownInput";
import { Comment, CommentCreateInput as CommentUpdateInputType } from "@shared/consts";
import { commentEndpoint } from "graphql/endpoints";

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
    session,
    zIndex,
}: CommentUpdateInputProps) => {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);
    const isLoggedIn = useMemo(() => Boolean(getCurrentUser(session).id), [session]);

    const [updateMutation, { loading: loadingAdd }] = useMutation<Comment, CommentUpdateInputType, 'commentUpdate'>(...commentEndpoint.update);
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
        validationSchema: commentValidation.update(),
        onSubmit: (values) => {
            // If not logged in, open login dialog
            //TODO
            mutationWrapper<Comment, CommentUpdateInputType>({
                mutation: updateMutation,
                input: shapeComment.update(comment, {
                    ...comment,
                    translations: values.translationsUpdate,
                }),
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

    const translations = useTranslatedFields({
        fields: ['text'],
        formik, 
        formikField: 'translationsUpdate', 
        language, 
        validationSchema: commentTranslationValidation.update(),
    });
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsUpdate', e, language);
    }, [formik, language]);

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
                <Typography component="h3" variant="h6" textAlign="left">Update comment</Typography>
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
                        errors={translations.errorsWithTranslations}
                        isCreate={true}
                        loading={formik.isSubmitting || loadingAdd}
                        onCancel={formik.resetForm}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.submitForm}
                    />
                </Grid>
            </Box>
        </form>
    )
}