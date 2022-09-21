import { Box, Button, CircularProgress, IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { CommentThreadItemProps } from '../types';
import { useCallback, useMemo, useState } from 'react';
import { TextLoading, UpvoteDownvote } from '../..';
import { displayDate, getTranslation, PubSub } from 'utils';
import { MarkdownInput } from 'components/inputs';
import {
    Delete as DeleteIcon,
    Reply as ReplyIcon,
} from '@mui/icons-material';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils';
import { DeleteOneType, ReportFor, StarFor, VoteFor } from '@shared/consts';
import { commentCreateForm as validationSchema } from '@shared/validation';
import { commentCreate, commentCreateVariables } from 'graphql/generated/commentCreate';
import { commentCreateMutation, deleteOneMutation } from 'graphql/mutation';
import { useFormik } from 'formik';
import { deleteOne, deleteOneVariables } from 'graphql/generated/deleteOne';
import { v4 as uuid } from 'uuid';
import { OwnerLabel } from 'components/text';
import { ShareButton } from 'components/buttons/ShareButton/ShareButton';
import { ReportButton, StarButton } from 'components/buttons';

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

    const { canDelete, canEdit, canReply, canReport, canStar, canVote, text } = useMemo(() => {
        const permissions = data?.permissionsComment;
        const languages = session?.languages ?? navigator.languages;
        return {
            canDelete: permissions?.canDelete === true,
            canEdit: permissions?.canEdit === true,
            canReply: permissions?.canReply === true,
            canReport: permissions?.canReport === true,
            canStar: permissions?.canStar === true,
            canVote: permissions?.canVote === true,
            text: getTranslation(data, 'text', languages, true),
        };
    }, [data, session]);

    const [replyOpen, setReplyOpen] = useState(false);
    const [addMutation, { loading: loadingAdd }] = useMutation<commentCreate, commentCreateVariables>(commentCreateMutation);
    const formik = useFormik({
        initialValues: {
            comment: '',
        },
        validationSchema,
        onSubmit: (values) => {
            if (!data) return;
            mutationWrapper({
                mutation: addMutation,
                input: {
                    id: uuid(),
                    createdFor: objectType,
                    forId: objectId,
                    parentId: data.id,
                    translationsCreate: [{
                        id: uuid(),
                        language,
                        text: values.comment,
                    }]
                },
                successCondition: (response) => response.data.commentCreate !== null,
                onSuccess: (response) => {
                    PubSub.get().publishSnack({ message: 'Comment created.', severity: 'success' });
                    formik.resetForm();
                    setReplyOpen(false);
                    handleCommentAdd(response.data.commentCreate);
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    const openReplyInput = useCallback(() => { setReplyOpen(true) }, []);
    const closeReplyInput = useCallback(() => {
        formik.resetForm();
        setReplyOpen(false)
    }, [formik]);

    /**
     * Handle add comment click
     */
    const handleReplySubmit = useCallback((event: any) => {
        // Make sure submit does not propagate past the form
        event.preventDefault();
        // Make sure form is valid
        if (!formik.isValid) return;
        // Submit form
        formik.submitForm();
    }, [formik]);

    const [deleteMutation, { loading: loadingDelete }] = useMutation<deleteOne, deleteOneVariables>(deleteOneMutation);
    const handleDelete = useCallback(() => {
        if (!data) return;
        // Confirmation dialog
        PubSub.get().publishAlertDialog({
            message: `Are you sure you want to delete this comment? This action cannot be undone.`,
            buttons: [
                {
                    text: 'Yes', onClick: () => {
                        mutationWrapper({
                            mutation: deleteMutation,
                            input: { id: data.id, objectType: DeleteOneType.Comment },
                            onSuccess: (response) => {
                                if (response?.data?.deleteOne?.success) {
                                    PubSub.get().publishSnack({ message: `Comment deleted.` });
                                    handleCommentRemove(data);
                                } else {
                                    PubSub.get().publishSnack({ message: `Error deleting comment.`, severity: 'error' });
                                }
                            },
                            onError: () => {
                                PubSub.get().publishSnack({ message: `Failed to delete comment.` });
                            }
                        })
                    }
                },
                { text: 'Cancel', onClick: () => { } },
            ]
        });
    }, [data, deleteMutation, handleCommentRemove]);

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
                        primary={text}
                    />)}
                    {/* Text buttons for reply, share, report, star, delete. */}
                    {isOpen && <Stack direction="row" spacing={1}>
                        {canVote && <UpvoteDownvote
                            direction="row"
                            session={session}
                            objectId={data?.id ?? ''}
                            voteFor={VoteFor.Comment}
                            isUpvoted={data?.isUpvoted}
                            score={data?.score}
                            onChange={() => { }}
                        />}
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
                                sx={{
                                    color: palette.background.textSecondary,
                                }}
                            >
                                <ReplyIcon />
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
                                sx={{
                                    color: palette.background.textSecondary,
                                }}
                            >
                                <DeleteIcon />
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
                                    value={formik.values.comment}
                                    minRows={3}
                                    onChange={(newText: string) => formik.setFieldValue('comment', newText)}
                                    error={formik.touched.comment && Boolean(formik.errors.comment)}
                                    helperText={formik.touched.comment ? formik.errors.comment as string : null}
                                />
                                <Stack direction="row" sx={{
                                    paddingTop: 1,
                                    display: 'flex',
                                    flexDirection: 'row-reverse',
                                }}>
                                    <Tooltip title={formik.errors.comment ? formik.errors.comment as string : ''}>
                                        <Button
                                            color="secondary"
                                            disabled={loadingAdd || formik.isSubmitting || !formik.isValid}
                                            onClick={handleReplySubmit}
                                            sx={{ marginLeft: 1 }}
                                        >
                                            {loadingAdd ? <CircularProgress size={24} /> : 'Add'}
                                        </Button>
                                    </Tooltip>
                                    <Button
                                        color="secondary"
                                        disabled={loadingAdd || formik.isSubmitting}
                                        onClick={closeReplyInput}
                                    >
                                        Cancel
                                    </Button>
                                </Stack>
                            </Box>
                        </form>
                    )}
                </Stack>
            </ListItem>
        </>
    )
}