import { ListItem, ListItemText, Stack, Typography, useTheme } from '@mui/material';
import { CommentListItemProps } from '../types';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation } from 'wouter';
import { TextLoading, UpvoteDownvote } from '../..';
import { displayDate, getCreatedByString, getTranslation, ObjectType, toCreatedBy } from 'utils';
import { owns } from 'utils/authentication';
import { LinkButton } from 'components/inputs';
import {
    Flag as ReportIcon,
    Reply as ReplyIcon,
    Share as ShareIcon,
    Star as IsStarredIcon,
    StarBorder as IsNotStarredIcon,
} from '@mui/icons-material';
import { useMutation } from '@apollo/client';
import { starMutation } from 'graphql/mutation';
import { star } from 'graphql/generated/star';
import { mutationWrapper } from 'graphql/utils';
import { VoteFor } from '@local/shared';

export function CommentListItem({
    data,
    isOpen,
    loading,
    session,
}: CommentListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { text } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            text: getTranslation(data, 'text', languages, true),
        };
    }, [data, session]);

    const ownedBy = useMemo<string | null>(() => getCreatedByString(data, session?.languages ?? navigator.languages), [data, session?.languages]);
    const toOwner = useCallback(() => { toCreatedBy(data, setLocation) }, [data, setLocation]);

    // Store changes made to comment (starred, voted, etc)
    const [changedData, setChangedData] = useState<CommentListItemProps['data'] | null>(null);
    useEffect(() => { setChangedData(data) }, [data]);

    // Handle star/unstar
    const [starMutate] = useMutation<star>(starMutation);
    const handleStarClick = useCallback((event: any) => {
        if (!session.id || !changedData) return;
        // Prevent propagation of normal click event
        event.stopPropagation();
        // Send star mutation
        mutationWrapper({
            mutation: starMutate,
            input: { isStar: !changedData?.isStarred, starFor: ObjectType.Comment, forId: data?.id ?? '' },
            onSuccess: (response) => {
                const isStarred = response.data.star;
                setChangedData({ ...changedData, isStarred, score: isStarred ? (data?.score ?? 0) + 1 : (data?.score ?? 0) - 1 })
            },
        })
    }, []);
    const StarIcon = changedData?.isStarred ? IsStarredIcon : IsNotStarredIcon;
    const starTooltip = changedData?.isStarred ? 'Remove from favorites' : 'Add to favorites';
    const starColor = session?.id ? '#cbae30' : 'rgb(189 189 189)';

    return (
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
                                {data?.role && <ListItemText
                                    primary={`(${data.role})`}
                                    sx={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        color: '#f2a7a7',
                                    }}
                                />}
                                {data?.creator?.id && data.creator.id === session?.id && <ListItemText
                                    primary={`(You)`}
                                    sx={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        color: '#f2a7a7',
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
                {/* Text buttons for reply, share, report, star, etc. */}
                {isOpen && <Stack direction="row" spacing={1}>
                    <UpvoteDownvote
                        direction="row"
                        session={session}
                        objectId={data?.id ?? ''}
                        voteFor={VoteFor.Comment}
                        isUpvoted={data?.isUpvoted}
                        score={data?.score}
                        onChange={() => { }}
                    />
                    {/* TODO */}
                </Stack>}
            </Stack>
        </ListItem>
    )
}