import { ListItem, ListItemText, Stack, useTheme } from '@mui/material';
import { CommentListItemProps } from '../types';
import { useCallback, useMemo } from 'react';
import { useLocation } from 'wouter';
import { TextLoading } from '../..';
import { getCreatedByString, getTranslation, toCreatedBy } from 'utils';
import { owns } from 'utils/authentication';
import { LinkButton } from 'components/inputs';

export function CommentListItem({
    data,
    loading,
    session,
    onVote,
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
                    {/* TODO */}
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