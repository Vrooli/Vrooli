import { useMutation } from '@apollo/client';
import { Box, Stack, Typography, useTheme } from '@mui/material';
import { DownvoteTallIcon, DownvoteWideIcon, UpvoteTallIcon, UpvoteWideIcon } from '@shared/icons';
import { voteEndpoint } from 'graphql/endpoints';
import { mutationWrapper } from 'graphql/utils';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { getCurrentUser } from 'utils/authentication';
import { UpvoteDownvoteProps } from '../types';

export const UpvoteDownvote = ({
    direction = "column",
    disabled = false,
    session,
    score,
    isUpvoted,
    objectId,
    voteFor,
    onChange,
}: UpvoteDownvoteProps) => {
    const { palette } = useTheme();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsUpvoted, setInternalIsUpvoted] = useState<boolean | null>(isUpvoted ?? null);
    useEffect(() => setInternalIsUpvoted(isUpvoted ?? null), [isUpvoted]);

    const internalScore = useMemo(() => {
        console.log('calculating internal score', score, isUpvoted, internalIsUpvoted);
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

    const [mutation] = useMutation(voteEndpoint.vote);
    const handleVote = useCallback((e: any, isUpvote: boolean | null, oldIsUpvote: boolean | null) => {
        // Prevent propagation of normal click event
        e.stopPropagation();
        // Send vote mutation
        mutationWrapper<vote_vote, voteVariables>({
            mutation,
            input: { isUpvote, voteFor, forId: objectId },
            onSuccess: (data) => { 
                // Determine new score
                let newScore: number = score ?? 0;
                // If vote is the same, score is the same
                if (isUpvote === oldIsUpvote) newScore += 0;
                // If both are not null, then score is changing by 2
                else if (isUpvote !== null && oldIsUpvote !== null) newScore += (isUpvote ? 2 : -2);
                // If original vote was null, then score is changing by 1
                else if (oldIsUpvote === null) newScore += (isUpvote ? 1 : -1);
                // If new vote is null, then score is changing by 1. This is the last case
                else newScore += (oldIsUpvote ? -1 : 1);
                onChange(isUpvote, newScore) 
            },
        })
    }, [mutation, voteFor, objectId, score, onChange]);

    const handleUpvoteClick = useCallback((event: any) => {
        if (!userId || disabled) return;
        // If already upvoted, cancel the vote
        const vote = internalIsUpvoted === true ? null : true;
        setInternalIsUpvoted(vote);
        handleVote(event, vote, internalIsUpvoted);
    }, [userId, disabled, internalIsUpvoted, handleVote]);

    const handleDownvoteClick = useCallback((event: any) => {
        if (!userId || disabled) return;
        // If already downvoted, cancel the vote
        const vote = internalIsUpvoted === false ? null : false;
        setInternalIsUpvoted(vote);
        handleVote(event, vote, internalIsUpvoted);
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