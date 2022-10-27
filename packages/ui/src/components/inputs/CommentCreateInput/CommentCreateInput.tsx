import { useMutation } from "@apollo/client";
import { DUMMY_ID, uuid } from "@shared/uuid";
import { commentTranslationCreate } from "@shared/validation";
import { CommentDialog } from "components/dialogs"
import { useCallback, useMemo } from "react";
import { getFormikErrorsWithTranslations, getTranslationData, handleTranslationBlur, handleTranslationChange, shapeCommentCreate, usePromptBeforeUnload, useWindowSize } from "utils";
import { CommentCreateInputProps } from "../types"
import { commentCreate as validationSchema } from '@shared/validation';
import { commentCreateMutation } from "graphql/mutation";
import { getCurrentUser } from "utils/authentication";
import { mutationWrapper } from "graphql/utils";
import { commentCreateVariables, commentCreate_commentCreate } from "graphql/generated/commentCreate";
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
        validationSchema,
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
                successMessage: () => 'Comment created',
                onSuccess: (data) => {
                    formik.resetForm();
                    onCommentAdd(data);
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty && formik.values.translationsCreate.some(t => t.text.trim().length > 0) });

    const { text, errorText, touchedText, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsCreate', language);
        return {
            text: value?.text ?? '',
            errorText: error?.text ?? '',
            touchedText: touched?.text ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsCreate', commentTranslationCreate),
        }
    }, [formik, language]);
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsCreate', e, language)
    }, [formik, language]);
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsCreate', e, language);
    }, [formik, language]);

    // If mobile, use CommentDialog
    if (isMobile) return (
        <CommentDialog
            errors={errors}
            errorText={errorText}
            handleClose={handleClose}
            isAdding={true}
            isOpen={true}
            language={language}
            onTranslationBlur={onTranslationBlur}
            onTranslationChange={onTranslationChange}
            parent={parent}
            text={text}
            touchedText={touchedText}
            zIndex={zIndex + 1}
        />
    )
    // Otherwise, use MarkdownInput with GridSubmitButtons
    return (
        <form>
            <Box sx={{ margin: 2 }}>
                <Typography component="h3" variant="h6" textAlign="left">Add comment</Typography>
                <MarkdownInput
                    id="add-comment"
                    placeholder="Please be nice to each other."
                    value={text}
                    minRows={3}
                    onChange={(newText: string) => onTranslationChange({ target: { name: 'text', value: newText } })}
                    error={touchedText && Boolean(errorText)}
                    helperText={touchedText ? errorText : null}
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