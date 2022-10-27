import { useMutation } from "@apollo/client";
import { DUMMY_ID } from "@shared/uuid";
import { commentTranslationUpdate } from "@shared/validation";
import { CommentDialog } from "components/dialogs"
import { useCallback, useMemo } from "react";
import { getFormikErrorsWithTranslations, getTranslationData, handleTranslationChange, shapeCommentUpdate, usePromptBeforeUnload, useWindowSize } from "utils";
import { CommentUpdateInputProps } from "../types"
import { commentCreate as validationSchema } from '@shared/validation';
import { commentUpdateMutation } from "graphql/mutation";
import { getCurrentUser } from "utils/authentication";
import { mutationWrapper } from "graphql/utils";
import { useFormik } from "formik";
import { Box, Grid, Typography, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons";
import { MarkdownInput } from "../MarkdownInput/MarkdownInput";
import { commentUpdateVariables, commentUpdate_commentUpdate } from "graphql/generated/commentUpdate";


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

    const [updateMutation, { loading: loadingAdd }] = useMutation(commentUpdateMutation);
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
        validationSchema,
        onSubmit: (values) => {
            // If not logged in, open login dialog
            //TODO
            mutationWrapper<commentUpdate_commentUpdate, commentUpdateVariables>({
                mutation: updateMutation,
                input: shapeCommentUpdate(comment as any, {
                    ...comment,
                    translations: values.translationsUpdate,
                } as any),
                successCondition: (data) => data !== null,
                successMessage: () => 'Comment updated',
                onSuccess: (data) => {
                    formik.resetForm();
                    onCommentUpdate(data);
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty && formik.values.translationsUpdate.some(t => t.text.trim().length > 0) });

    const { text, errorText, errors } = useMemo(() => {
        const { error, value } = getTranslationData(formik, 'translationsUpdate', language);
        return {
            text: value?.text ?? '',
            errorText: error?.text ?? '',
            errors: getFormikErrorsWithTranslations(formik, 'translationsUpdate', commentTranslationUpdate),
        }
    }, [formik, language]);
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsUpdate', e, language);
    }, [formik, language]);

    // If mobile, use CommentDialog
    if (isMobile) return (
        <CommentDialog
            errorText={errorText}
            handleClose={handleClose}
            handleSubmit={formik.handleSubmit}
            isAdding={false}
            isOpen={true}
            language={language}
            onTranslationChange={onTranslationChange}
            parent={parent}
            text={text}
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
                    value={text}
                    minRows={3}
                    onChange={(newText: string) => onTranslationChange({ target: { name: 'text', value: newText } })}
                    error={text.length > 0 && Boolean(errorText)}
                    helperText={text.length > 0 ? errorText : ''}
                />
                <Grid container spacing={1} sx={{
                    width: 'min(100%, 400px)',
                    marginLeft: 'auto',
                    marginTop: 1,
                }}>
                    <GridSubmitButtons
                        disabledSubmit={!isLoggedIn}
                        errors={errors}
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