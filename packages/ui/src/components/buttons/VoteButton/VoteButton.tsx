import { Box, Stack, Typography, useTheme } from '@mui/material';
import { Success } from '@shared/consts';
import { DownvoteTallIcon, DownvoteWideIcon, UpvoteTallIcon, UpvoteWideIcon } from '@shared/icons';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { ObjectActionComplete } from 'utils/actions/objectActions';
import { getCurrentUser } from 'utils/authentication/session';
import { useVoter } from 'utils/hooks/useVoter';
import { VoteButtonProps } from '../types';

export const VoteButton = ({
    direction = "column",
    disabled = false,
    session,
    score,
    isUpvoted,
    objectId,
    voteFor,
    onChange,
}: VoteButtonProps) => {
    const { palette } = useTheme();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsUpvoted, setInternalIsUpvoted] = useState<boolean | null>(isUpvoted ?? null);
    useEffect(() => setInternalIsUpvoted(isUpvoted ?? null), [isUpvoted]);

    const internalScore = useMemo(() => {
        if (isUpvoted) console.log('calculating internal score', score, isUpvoted, internalIsUpvoted);
        const scoreNum = score ?? 0;
        // If the score and internal score match, return the score
        if (internalIsUpvoted === isUpvoted) return scoreNum;
        // Otherwise, determine score based on internal state
        if ((isUpvoted === true && internalIsUpvoted === null) ||
            (isUpvoted === null && internalIsUpvoted === false)) return scoreNum - 1;
        if ((isUpvoted === false && internalIsUpvoted === null) ||
            (isUpvoted === null && internalIsUpvoted === true)) return scoreNum + 1;
        return scoreNum;
    }, [internalIsUpvoted, isUpvoted, score]);

    const onVoteComplete = useCallback((action: ObjectActionComplete.VoteDown | ObjectActionComplete.VoteUp, data: Success) => {
        const isUpvoted = action === ObjectActionComplete.VoteUp;
        const wasUpvoted = internalIsUpvoted;
        // Determine new score
        let newScore: number = score ?? 0;
        // If vote is the same, score is the same
        if (isUpvoted === wasUpvoted) newScore += 0;
        // If both are not null, then score is changing by 2
        else if (isUpvoted !== null && wasUpvoted !== null) newScore += (isUpvoted ? 2 : -2);
        // If original vote was null, then score is changing by 1
        else if (wasUpvoted === null) newScore += (isUpvoted ? 1 : -1);
        // If new vote is null, then score is changing by 1. This is the last case
        else newScore += (wasUpvoted ? -1 : 1);
        onChange(isUpvoted, newScore) 
    }, [internalIsUpvoted, onChange, score]);

    const { handleVote: mutate } = useVoter({
        objectId,
        objectType: voteFor,
        onActionComplete: onVoteComplete,
    });

    const handleVote = useCallback((event: any, isUpvote: boolean | null) => {
        // Prevent propagation of normal click event
        event.stopPropagation();
        event.preventDefault();
        // Send vote mutation
        mutate(isUpvote);
    }, [mutate]);

    const handleUpvoteClick = useCallback((event: any) => {
        if (!userId || disabled) return;
        // If already upvoted, cancel the vote
        const vote = internalIsUpvoted === true ? null : true;
        setInternalIsUpvoted(vote);
        handleVote(event, vote);
    }, [userId, disabled, internalIsUpvoted, handleVote]);

    const handleDownvoteClick = useCallback((event: any) => {
        if (!userId || disabled) return;
        // If already downvoted, cancel the vote
        const vote = internalIsUpvoted === false ? null : false;
        setInternalIsUpvoted(vote);
        handleVote(event, vote);
    }, [userId, disabled, internalIsUpvoted, handleVote]);

    const { UpvoteIcon, upvoteColor } = useMemo(() => {
        const upvoteColor = (!userId || disabled) ? palette.background.textSecondary :
            internalIsUpvoted === true ? "#34c38b" :
                "#687074";
        const UpvoteIcon = direction === "column" ? UpvoteWideIcon : UpvoteTallIcon;
        return { UpvoteIcon, upvoteColor };
    }, [userId, disabled, palette.background.textSecondary, internalIsUpvoted, direction]);

    const { DownvoteIcon, downvoteColor } = useMemo(() => {
        const downvoteColor = (!userId || disabled) ? palette.background.textSecondary :
            internalIsUpvoted === false ? "#af2929" :
                "#687074";
        const DownvoteIcon = direction === "column" ? DownvoteWideIcon : DownvoteTallIcon;
        return { DownvoteIcon, downvoteColor };
    }, [userId, disabled, palette.background.textSecondary, internalIsUpvoted, direction]);

    return (
        <Stack direction={direction} sx={{ pointerEvents: 'none' }}>
            {/* Upvote arrow */}
            <Box
                display="inline-block"
                onClick={handleUpvoteClick}
                role="button"
                aria-pressed={internalIsUpvoted === true}
                sx={{
                    cursor: (userId || disabled) ? 'pointer' : 'default',
                    pointerEvents: 'all',
                    display: 'flex',
                    '&:hover': {
                        filter: userId ? `brightness(120%)` : 'none',
                        transition: 'filter 0.2s',
                    },
                }}
            >
                <UpvoteIcon width="36px" height="36px" fill={upvoteColor} />
            </Box>
            {/* Score */}
            <Typography variant="body1" textAlign="center" sx={{ margin: 'auto', pointerEvents: 'none' }}>{internalScore}</Typography>
            {/* Downvote arrow */}
            <Box
                display="inline-block"
                onClick={handleDownvoteClick}
                role="button"
                aria-pressed={internalIsUpvoted === false}
                sx={{
                    cursor: (userId || disabled) ? 'pointer' : 'default',
                    pointerEvents: 'all',
                    display: 'flex',
                    '&:hover': {
                        filter: userId ? `brightness(120%)` : 'none',
                        transition: 'filter 0.2s',
                    },
                }}
            >
                <DownvoteIcon width="36px" height="36px" fill={downvoteColor} />
            </Box>
        </Stack>
    )
}