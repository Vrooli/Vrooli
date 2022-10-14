import { Box, Grid, IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { CommentThreadItemProps } from '../types';
import { useCallback, useMemo, useState } from 'react';
import { TextLoading, UpvoteDownvote } from '../..';
import { displayDate, getFormikErrorsWithTranslations, getTranslation, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, PubSub, usePromptBeforeUnload } from 'utils';
import { MarkdownInput } from 'components/inputs';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils';
import { DeleteOneType, ReportFor, StarFor, VoteFor } from '@shared/consts';
import { commentCreate as validationSchema, commentTranslationCreate } from '@shared/validation';
import { commentCreate, commentCreateVariables, commentCreate_commentCreate } from 'graphql/generated/commentCreate';
import { commentCreateMutation, deleteOneMutation } from 'graphql/mutation';
import { useFormik } from 'formik';
import { deleteOneVariables, deleteOne_deleteOne } from 'graphql/generated/deleteOne';
import { DUMMY_ID, uuid } from '@shared/uuid';
import { OwnerLabel } from 'components/text';
import { ShareButton } from 'components/buttons/ShareButton/ShareButton';
import { GridSubmitButtons, ReportButton, StarButton } from 'components/buttons';
import { DeleteIcon, ReplyIcon } from '@shared/icons';
import { uuidValidate } from '@shared/uuid';

export function CommentThreadItem({
    data,
    handleCommentAdd,
    handleCommentRemove,
    isOpen,
    language,
    loading,
    objectId,
    objectType,
    session,
    zIndex,
}: CommentThreadItemProps) {
    const { palette } = useTheme();

    const { canDelete, canEdit, canReply, canReport, canStar, canVote, displayText } = useMemo(() => {
        const permissions = data?.permissionsComment;
        const languages = getUserLanguages(session);
        return {
            canDelete: permissions?.canDelete === true,
            canEdit: permissions?.canEdit === true,
            canReply: permissions?.canReply === true,
            canReport: permissions?.canReport === true,
            canStar: permissions?.canStar === true,
            canVote: permissions?.canVote === true,
            displayText: getTranslation(data, 'text', languages, true),
        };
    }, [data, session]);

    const [replyOpen, setReplyOpen] = useState(false);
    const [addMutation, { loading: loadingAdd }] = useMutation<commentCreate, commentCreateVariables>(commentCreateMutation);
    const formik = useFormik({
        initialValues: {
            id: DUMMY_ID,
            createdFor: objectType,
            forId: objectId,
            parentId: data?.id,
            translationsCreate: [{
                id: DUMMY_ID,
                language,
                text: '',
            }],
        },
        validationSchema,
        enableReinitialize: true,
        onSubmit: (values) => {
            if (!data) return;
            mutationWrapper<commentCreate_commentCreate, commentCreateVariables>({
                mutation: addMutation,
                input: {
                    id: uuid(),
                    createdFor: values.createdFor,
                    forId: values.forId,
                    parentId: values.parentId,
                    translationsCreate: values.translationsCreate.map(t => ({
                        ...t,
                        id: t.id === DUMMY_ID ? uuid() : t.id,
                    })),
                },
                successCondition: (data) => data !== null,
                successMessage: () => 'Comment created.',
                onSuccess: (data) => {
                    formik.resetForm();
                    setReplyOpen(false);
                    handleCommentAdd(data);
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Current text, as well as errors
    const { text, errorText, touchedText, errors } = useMemo(() => {
        console.log('comment threaditem gettransdata')
        const { error, touched, value } = getTranslationData(formik, 'translationsCreate', language);
        return {
            text: value?.text ?? '',
            errorText: error?.text ?? '',
            touchedText: touched?.text ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsCreate', commentTranslationCreate),
        }
    }, [formik, language]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsCreate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsCreate', e, language)
    }, [formik, language]);

    const openReplyInput = useCallback(() => { setReplyOpen(true) }, []);
    const closeReplyInput = useCallback(() => {
        formik.resetForm();
        setReplyOpen(false)
    }, [formik]);

    const [deleteMutation, { loading: loadingDelete }] = useMutation(deleteOneMutation);
    const handleDelete = useCallback(() => {
        if (!data) return;
        // Confirmation dialog
        PubSub.get().publishAlertDialog({
            message: `Are you sure you want to delete this comment? This action cannot be undone.`,
            buttons: [
                {
                    text: 'Yes', onClick: () => {
                        mutationWrapper<deleteOne_deleteOne, deleteOneVariables>({
                            mutation: deleteMutation,
                            input: { id: data.id, objectType: DeleteOneType.Comment },
                            successCondition: (data) => data.success,
                            successMessage: () => 'Comment deleted.',
                            onSuccess: () => {
                                handleCommentRemove(data);
                            },
                            errorMessage: () => 'Failed to delete comment.',
                        })
                    }
                },
                { text: 'Cancel', onClick: () => { } },
            ]
        });
    }, [data, deleteMutation, handleCommentRemove]);

    const isLoggedIn = useMemo(() => session?.isLoggedIn === true && uuidValidate(session?.id ?? ''), [session]);

    return (
        <>
            <ListItem
                id={`comment-${data?.id}`}
                disablePadding
                sx={{
                    display: 'flex',
                    background: 'transparent',
                }}
            >
                <Stack
                    direction="column"
                    spacing={1}
                    pl={2}
                    sx={{
                        width: '-webkit-fill-available',
                        display: 'grid',
                    }}
                >
                    {/* Username and time posted */}
                    <Stack direction="row" spacing={1}>
                        {/* Username and role */}
                        {
                            <Stack direction="row" spacing={1} sx={{
                                overflow: 'auto',
                            }}>
                                <OwnerLabel
                                    objectType={objectType}
                                    owner={data?.creator}
                                    session={session}
                                    sxs={{
                                        label: {
                                            color: palette.background.textPrimary,
                                            fontWeight: 'bold',
                                        }
                                    }} />
                                {canEdit && !(data?.creator?.id && data.creator.id === session?.id) && <ListItemText
                                    primary={`(Can Edit)`}
                                    sx={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        color: palette.mode === 'light' ? '#fa4f4f' : '#f2a7a7',
                                    }}
                                />}
                                {data?.creator?.id && data.creator.id === session?.id && <ListItemText
                                    primary={`(You)`}
                                    sx={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        color: palette.mode === 'light' ? '#fa4f4f' : '#f2a7a7',
                                    }}
                                />}
                            </Stack>
                        }
                        {/* Time posted */}
                        <ListItemText
                            primary={displayDate(data?.created_at, false)}
                            sx={{
                                display: 'flex',
                                alignItems: 'center',
                            }}
                        />
                    </Stack>
                    {/* Text */}
                    {isOpen && (loading ? <TextLoading /> : <ListItemText
                        primary={displayText}
                    />)}
                    {/* Text buttons for reply, share, report, star, delete. */}
                    {isOpen && <Stack direction="row" spacing={1}>
                        <UpvoteDownvote
                            direction="row"
                            disabled={!canVote}
                            session={session}
                            objectId={data?.id ?? ''}
                            voteFor={VoteFor.Comment}
                            isUpvoted={data?.isUpvoted}
                            score={data?.score}
                            onChange={() => { }}
                        />
                        {canStar && <StarButton
                            session={session}
                            objectId={data?.id ?? ''}
                            starFor={StarFor.Comment}
                            isStar={data?.isStarred ?? false}
                            showStars={false}
                            tooltipPlacement="top"
                        />}
                        {canReply && <Tooltip title="Reply" placement='top'>
                            <IconButton
                                onClick={openReplyInput}
                            >
                                <ReplyIcon fill={palette.background.textSecondary} />
                            </IconButton>
                        </Tooltip>}
                        <ShareButton objectType={objectType} zIndex={zIndex} />
                        {canReport && <ReportButton
                            forId={data?.id ?? ''}
                            reportFor={objectType as any as ReportFor}
                            session={session}
                            zIndex={zIndex}
                        />}
                        {canDelete && <Tooltip title="Delete" placement='top'>
                            <IconButton
                                onClick={handleDelete}
                                disabled={loadingDelete}
                            >
                                <DeleteIcon fill={palette.background.textSecondary} />
                            </IconButton>
                        </Tooltip>}
                    </Stack>}
                    {/* New reply input */}
                    {replyOpen && (
                        <form>
                            <Box sx={{ margin: 2 }}>
                                <MarkdownInput
                                    id={`add-reply-${data?.id}`}
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
                                        disabledCancel={formik.isSubmitting}
                                        disabledSubmit={!isLoggedIn}
                                        errors={errors}
                                        isCreate={true}
                                        loading={formik.isSubmitting || loadingAdd}
                                        onCancel={closeReplyInput}
                                        onSetSubmitting={formik.setSubmitting}
                                        onSubmit={formik.submitForm}
                                    />
                                </Grid>
                            </Box>
                        </form>
                    )}
                </Stack>
            </ListItem>
        </>
    )
}