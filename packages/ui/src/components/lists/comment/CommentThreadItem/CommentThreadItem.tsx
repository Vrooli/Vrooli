import { IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { BookmarkFor, CommentFor, DeleteOneInput, DeleteType, ReportFor, Success, VoteFor } from '@shared/consts';
import { DeleteIcon, ReplyIcon } from '@shared/icons';
import { deleteOneOrManyDeleteOne } from 'api/generated/endpoints/deleteOneOrMany_deleteOne';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { BookmarkButton } from 'components/buttons/BookmarkButton/BookmarkButton';
import { ReportButton } from 'components/buttons/ReportButton/ReportButton';
import { ShareButton } from 'components/buttons/ShareButton/ShareButton';
import { VoteButton } from 'components/buttons/VoteButton/VoteButton';
import { CommentCreateInput } from 'components/inputs/CommentCreateInput/CommentCreateInput';
import { CommentUpdateInput } from 'components/inputs/CommentUpdateInput/CommentUpdateInput';
import { TextLoading } from 'components/lists/TextLoading/TextLoading';
import { OwnerLabel } from 'components/text/OwnerLabel/OwnerLabel';
import { useCallback, useContext, useMemo, useState } from 'react';
import { getCurrentUser } from 'utils/authentication/session';
import { getYou } from 'utils/display/listTools';
import { displayDate } from 'utils/display/stringTools';
import { getTranslation, getUserLanguages } from 'utils/display/translationTools';
import { ObjectType } from 'utils/navigation/openObject';
import { PubSub } from 'utils/pubsub';
import { SessionContext } from 'utils/SessionContext';
import { CommentThreadItemProps } from '../types';

export function CommentThreadItem({
    data,
    handleCommentAdd,
    handleCommentRemove,
    handleCommentUpdate,
    isOpen,
    language,
    loading,
    object,
    zIndex,
}: CommentThreadItemProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();

    const { objectId, objectType } = useMemo(() => ({
        objectId: object?.id,
        objectType: object?.__typename as CommentFor,
    }), [object]);
    const { isBookmarked, isUpvoted } = useMemo(() => getYou(object as any), [object]);

    const { canDelete, canUpdate, canReply, canReport, canBookmark, canVote, displayText } = useMemo(() => {
        const { canDelete, canUpdate, canReply, canReport, canBookmark, canVote } = data?.you ?? {};
        const languages = getUserLanguages(session);
        const { text } = getTranslation(data, languages, true);
        return { canDelete, canUpdate, canReply, canReport, canBookmark, canVote, displayText: text };
    }, [data, session]);

    const [deleteMutation, { loading: loadingDelete }] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne);
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
                                    sxs={{
                                        label: {
                                            color: palette.background.textPrimary,
                                            fontWeight: 'bold',
                                        }
                                    }} />}
                                {canUpdate && !(data?.owner?.id && data.owner.id === getCurrentUser(session).id) && <ListItemText
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
                        <VoteButton
                            direction="row"
                            disabled={!canVote}
                            objectId={data?.id ?? ''}
                            voteFor={VoteFor.Comment}
                            isUpvoted={isUpvoted}
                            score={data?.score}
                            onChange={() => { }}
                        />
                        {canBookmark && <BookmarkButton
                            objectId={data?.id ?? ''}
                            bookmarkFor={BookmarkFor.Comment}
                            isBookmarked={isBookmarked ?? false}
                            showBookmarks={false}
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
                            zIndex={zIndex}
                        />
                    }
                </Stack>
            </ListItem>
        </>
    )
}