import { useMutation } from "@apollo/client";
import { DUMMY_ID, uuid } from "@shared/uuid";
import { CommentDialog } from "components/dialogs"
import { useCallback, useMemo } from "react";
import { getFormikErrorsWithTranslations, getTranslationData, handleTranslationChange, shapeCommentCreate, usePromptBeforeUnload, useWindowSize } from "utils";
import { CommentCreateInputProps } from "../types"
import { commentValidation, commentTranslationValidation } from '@shared/validation';
import { getCurrentUser } from "utils/authentication";
import { mutationWrapper } from "graphql/utils";
import { useFormik } from "formik";
import { Box, Grid, Typography, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons";
import { MarkdownInput } from "../MarkdownInput/MarkdownInput";


/**
 * MarkdownInput/CommentContainer wrapper for creating comments
 */
export const CommentCreateInput = ({
    handleClose,
    language,
    objectId,
    objectType,
    onCommentAdd,
    parent,
    session,
    zIndex,
}: CommentCreateInputProps) => {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);
    const isLoggedIn = useMemo(() => Boolean(getCurrentUser(session).id), [session]);

    const [addMutation, { loading: loadingAdd }] = useMutation(commentCreateMutation);
    const formik = useFormik({
        initialValues: {
            id: DUMMY_ID,
            createdFor: objectType,
            forId: objectId,
            parentId: parent?.id ?? undefined,
            translationsCreate: [{
                id: DUMMY_ID,
                language,
                text: '',
            }],
        },
        validationSchema: commentValidation.create,
        onSubmit: (values) => {
            // If not logged in, open login dialog
            //TODO
            mutationWrapper<commentCreate_commentCreate, commentCreateVariables>({
                mutation: addMutation,
                input: shapeCommentCreate({
                    id: uuid(),
                    commentedOn: { __typename: values.createdFor, id: values.forId },
                    translations: values.translationsCreate,
                }, values.parentId),
                successCondition: (data) => data !== null,
                successMessage: () => ({ key: 'CommentCreated' }),
                onSuccess: (data) => {
                    formik.resetForm();
                    onCommentAdd(data);
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty && formik.values.translationsCreate.some(t => t.text.trim().length > 0) });

    const { text, errorText, errors } = useMemo(() => {
        const { error, value } = getTranslationData(formik, 'translationsCreate', language);
        return {
            text: value?.text ?? '',
            errorText: error?.text ?? '',
            errors: getFormikErrorsWithTranslations(formik, 'translationsCreate', commentTranslationCreate),
        }
    }, [formik, language]);
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsCreate', e, language);
    }, [formik, language]);

    // If mobile, use CommentDialog
    if (isMobile) return (
        <CommentDialog
            errorText={errorText}
            handleClose={handleClose}
            handleSubmit={formik.handleSubmit}
            isAdding={true}
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
                <Typography component="h3" variant="h6" textAlign="left">Add comment</Typography>
                <MarkdownInput
                    id={`add-comment-${parent?.id ?? 'root'}`}
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