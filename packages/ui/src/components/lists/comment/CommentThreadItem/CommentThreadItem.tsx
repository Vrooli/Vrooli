import { Box, Button, CircularProgress, IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { CommentThreadItemProps } from '../types';
import { useCallback, useMemo, useState } from 'react';
import { useLocation } from '@local/route';
import { StarButton, TextLoading, UpvoteDownvote } from '../..';
import { displayDate, getCreatedByString, getTranslation, PubSub, toCreatedBy } from 'utils';
import { LinkButton, MarkdownInput } from 'components/inputs';
import {
    Delete as DeleteIcon,
    Flag as ReportIcon,
    Reply as ReplyIcon,
    Share as ShareIcon,
} from '@mui/icons-material';
import { useMutation } from '@apollo/client';
import { mutationWrapper } from 'graphql/utils';
import { DeleteOneType, ReportFor, StarFor, VoteFor } from '@local/shared';
import { commentCreateForm as validationSchema } from '@local/shared';
import { commentCreate, commentCreateVariables } from 'graphql/generated/commentCreate';
import { commentCreateMutation, deleteOneMutation } from 'graphql/mutation';
import { useFormik } from 'formik';
import { deleteOne, deleteOneVariables } from 'graphql/generated/deleteOne';
import { ReportDialog } from 'components/dialogs';
import { v4 as uuid } from 'uuid';

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
    const [, setLocation] = useLocation();

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

    const ownedBy = useMemo<string | null>(() => getCreatedByString(data, session?.languages ?? navigator.languages), [data, session?.languages]);
    const toOwner = useCallback(() => { toCreatedBy(data, setLocation) }, [data, setLocation]);

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

    const handleShare = useCallback(() => {
        //TODO
    }, []);

    const [reportOpen, setReportOpen] = useState<boolean>(false);
    const openReport = useCallback(() => setReportOpen(true), [setReportOpen]);
    const closeReport = useCallback(() => setReportOpen(false), [setReportOpen]);

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
            <ReportDialog
                forId={data?.id ?? ''}
                onClose={closeReport}
                open={reportOpen}
                reportFor={ReportFor.Comment}
                session={session}
                zIndex={zIndex + 1}
            />
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
                            ownedBy && (
                                <Stack direction="row" spacing={1} sx={{
                                    overflow: 'auto',
                                }}>
                                    <LinkButton
                                        onClick={toOwner}
                                        text={ownedBy}
                                        sxs={{
                                            text: {
                                                color: palette.background.textPrimary,
                                                fontWeight: 'bold',
                                            }
                                        }}
                                    />
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
                            )
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
                                    color: palette.background.textPrimary,
                                }}
                            >
                                <ReplyIcon />
                            </IconButton>
                        </Tooltip>}
                        <Tooltip title="Share" placement='top'>
                            <IconButton
                                onClick={handleShare}
                                sx={{
                                    color: palette.background.textPrimary,
                                }}
                            >
                                <ShareIcon />
                            </IconButton>
                        </Tooltip>
                        {canReport && <Tooltip title="Report" placement='top'>
                            <IconButton
                                onClick={openReport}
                                sx={{
                                    color: palette.background.textPrimary,
                                }}
                            >
                                <ReportIcon />
                            </IconButton>
                        </Tooltip>}
                        {canDelete && <Tooltip title="Delete" placement='top'>
                            <IconButton
                                onClick={handleDelete}
                                disabled={loadingDelete}
                                sx={{
                                    color: palette.background.textPrimary,
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