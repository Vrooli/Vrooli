import { Box, Grid, Typography, useTheme } from "@mui/material";
import { Comment, CommentCreateInput as CommentCreateInputType } from "@shared/consts";
import { DUMMY_ID, uuid } from "@shared/uuid";
import { commentTranslationValidation, commentValidation } from '@shared/validation';
import { commentCreate } from "api/generated/endpoints/comment_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { CommentDialog } from "components/dialogs/CommentDialog/CommentDialog";
import { useFormik } from "formik";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { checkIfLoggedIn } from "utils/authentication/session";
import { getUserLanguages } from "utils/display/translationTools";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { shapeComment } from "utils/shape/models/comment";
import { MarkdownInput } from "../MarkdownInput/MarkdownInput";
import { CommentCreateInputProps } from "../types";

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
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);
    const isLoggedIn = useMemo(() => checkIfLoggedIn(session), [session]);

    const [addMutation, { loading: loadingAdd }] = useCustomMutation<Comment, CommentCreateInputType>(commentCreate);
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
        validationSchema: commentValidation.create({}),
        onSubmit: (values) => {
            // If not logged in, open login dialog
            //TODO
            mutationWrapper<Comment, CommentCreateInputType>({
                mutation: addMutation,
                input: shapeComment.create({
                    id: uuid(),
                    commentedOn: { __typename: values.createdFor, id: values.forId },
                    threadId: parent?.id ?? null,
                    translations: values.translationsCreate,
                }),
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

    const {
        onTranslationChange,
        setLanguage,
        translations,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['text'],
        formik,
        formikField: 'translationsCreate',
        validationSchema: commentTranslationValidation.create({}),
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
            isAdding={true}
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
                <Typography component="h3" variant="h6" textAlign="left">{t('AddComment')}</Typography>
                <MarkdownInput
                    id={`add-comment-${parent?.id ?? 'root'}`}
                    placeholder={t(`PleaseBeNice`)}
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
                        display="dialog"
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