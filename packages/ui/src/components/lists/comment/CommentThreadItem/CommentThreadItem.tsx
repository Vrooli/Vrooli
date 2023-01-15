import { IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { CommentThreadItemProps } from '../types';
import { useCallback, useMemo, useState } from 'react';
import { TextLoading, UpvoteDownvote } from '../..';
import { displayDate, getTranslation, getUserLanguages, getYou, ObjectType, PubSub } from 'utils';
import { CommentCreateInput } from 'components/inputs';
import { useMutation } from 'graphql/hooks';
import { mutationWrapper } from 'graphql/utils';
import { CommentFor, DeleteOneInput, DeleteType, ReportFor, StarFor, Success, VoteFor } from '@shared/consts';
import { OwnerLabel } from 'components/text';
import { ShareButton } from 'components/buttons/ShareButton/ShareButton';
import { ReportButton, StarButton } from 'components/buttons';
import { DeleteIcon, ReplyIcon } from '@shared/icons';
import { CommentUpdateInput } from 'components/inputs/CommentUpdateInput/CommentUpdateInput';
import { getCurrentUser } from 'utils/authentication';
import { deleteOneOrManyEndpoint } from 'graphql/endpoints';

export function CommentThreadItem({
    data,
    handleCommentAdd,
    handleCommentRemove,
    handleCommentUpdate,
    isOpen,
    language,
    loading,
    object,
    session,
    zIndex,
}: CommentThreadItemProps) {
    const { palette } = useTheme();

    const { objectId, objectType } = useMemo(() => ({
        objectId: object?.id,
        objectType: object?.__typename as CommentFor,
    }), [object]);
    const { isStarred, isUpvoted } = useMemo(() => getYou(object as any), [object]);

    const { canDelete, canEdit, canReply, canReport, canStar, canVote, displayText } = useMemo(() => {
        const { canDelete, canEdit, canReply, canReport, canStar, canVote } = data?.you ?? {};
        const languages = getUserLanguages(session);
        const { text } = getTranslation(data, languages, true);
        return { canDelete, canEdit, canReply, canReport, canStar, canVote, displayText: text };
    }, [data, session]);

    const [deleteMutation, { loading: loadingDelete }] = useMutation<Success, DeleteOneInput, 'deleteOne'>(...deleteOneOrManyEndpoint.deleteOne);
    const handleDelete = useCallback(() => {
        if (!data) return;
        // Confirmation dialog
        PubSub.get().publishAlertDialog({
            messageKey: 'DeleteCommentConfirm',
            buttons: [
                {
                    labelKey: 'Yes', onClick: () => {
                        mutationWrapper<Success, DeleteOneInput>({
                            mutation: deleteMutation,
                            input: { id: data.id, objectType: DeleteType.Comment },
                            successCondition: (data) => data.success,
                            successMessage: () => ({ key: 'CommentDeleted' }),
                            onSuccess: () => {
                                handleCommentRemove(data);
                            },
                            errorMessage: () => ({ key: 'DeleteCommentFailed' }),
                        })
                    }
                },
                { labelKey: 'Cancel' },
            ]
        });
    }, [data, deleteMutation, handleCommentRemove]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState<boolean>(false);
    const [commentToUpdate, setCommentToUpdate] = useState<Comment | null>(null);
    const handleAddCommentOpen = useCallback(() => setIsAddCommentOpen(true), []);
    const handleAddCommentClose = useCallback(() => setIsAddCommentOpen(false), []);
    const handleUpdateCommentOpen = useCallback((comment: Comment) => { setCommentToUpdate(comment) }, []);
    const handleUpdateCommentClose = useCallback(() => { setCommentToUpdate(null) }, []);

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
                                {objectType && <OwnerLabel
                                    objectType={objectType as unknown as ObjectType}
                                    owner={data?.owner}
                                    session={session}
                                    sxs={{
                                        label: {
                                            color: palette.background.textPrimary,
                                            fontWeight: 'bold',
                                        }
                                    }} />}
                                {canEdit && !(data?.owner?.id && data.owner.id === getCurrentUser(session).id) && <ListItemText
                                    primary={`(Can Edit)`}
                                    sx={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        color: palette.mode === 'light' ? '#fa4f4f' : '#f2a7a7',
                                    }}
                                />}
                                {data?.owner?.id && data.owner.id === getCurrentUser(session).id && <ListItemText
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
                            isUpvoted={isUpvoted}
                            score={data?.score}
                            onChange={() => { }}
                        />
                        {canStar && <StarButton
                            session={session}
                            objectId={data?.id ?? ''}
                            starFor={StarFor.Comment}
                            isStar={isStarred ?? false}
                            showStars={false}
                            tooltipPlacement="top"
                        />}
                        {canReply && <Tooltip title="Reply" placement='top'>
                            <IconButton
                                onClick={handleAddCommentOpen}
                            >
                                <ReplyIcon fill={palette.background.textSecondary} />
                            </IconButton>
                        </Tooltip>}
                        <ShareButton object={object} zIndex={zIndex} />
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
                    {/* Add comment */}
                    {
                        isAddCommentOpen && objectId && objectType && <CommentCreateInput
                            handleClose={handleAddCommentClose}
                            language={language}
                            objectId={objectId}
                            objectType={objectType}
                            onCommentAdd={handleCommentAdd}
                            parent={(object as any) ?? null}
                            session={session}
                            zIndex={zIndex}
                        />
                    }
                    {/* Update comment */}
                    {
                        commentToUpdate && objectId && objectType && <CommentUpdateInput
                            comment={commentToUpdate as any}
                            handleClose={handleUpdateCommentClose}
                            language={language}
                            objectId={objectId}
                            objectType={objectType}
                            onCommentUpdate={handleCommentUpdate}
                            parent={(object as any) ?? null}
                            session={session}
                            zIndex={zIndex}
                        />
                    }
                </Stack>
            </ListItem>
        </>
    )
}