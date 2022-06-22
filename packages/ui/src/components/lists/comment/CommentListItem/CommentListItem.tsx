import { ListItem, ListItemText, Stack, Typography, useTheme } from '@mui/material';
import { CommentListItemProps } from '../types';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation } from 'wouter';
import { TextLoading } from '../..';
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

export function CommentListItem({
    data,
    loading,
    session,
}: CommentListItemProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const canEdit = useMemo<boolean>(() => owns(data?.role), [data]);
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
                    {/* Username */}
                    {ownedBy && (
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
                    )}
                    {/* Time posted */}
                    <Typography variant="body2">
                        {displayDate(data?.created_at, false)}
                    </Typography>
                </Stack>
                {/* Text */}
                {loading ? <TextLoading /> : <ListItemText
                    primary={text}
                />}
                {/* Text buttons for reply, share, report, star, etc. */}
                {/* TODO */}
            </Stack>
            {/* <UpvoteDownvote
                session={session}
                objectId={data?.id ?? ''}
                voteFor={objectType as VoteFor}
                isUpvoted={data?.isUpvoted}
                score={data?.score}
                onChange={onVote}
            /> */}
        </ListItem>
    )
}