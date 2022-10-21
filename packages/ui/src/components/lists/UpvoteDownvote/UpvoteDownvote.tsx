import { useMutation } from '@apollo/client';
import { Box, Stack, Typography } from '@mui/material';
import { DownvoteTallIcon, DownvoteWideIcon, UpvoteTallIcon, UpvoteWideIcon } from '@shared/icons';
import { voteVariables, vote_vote } from 'graphql/generated/vote';
import { voteMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
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
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsUpvoted, setInternalIsUpvoted] = useState<boolean | null>(isUpvoted ?? null);
    useEffect(() => setInternalIsUpvoted(isUpvoted ?? null), [isUpvoted]);

    const internalScore = useMemo(() => {
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

    const [mutation] = useMutation(voteMutation);
    const handleVote = useCallback((e: any, isUpvote: boolean | null) => {
        // Prevent propagation of normal click event
        e.stopPropagation();
        // Send vote mutation
        mutationWrapper<vote_vote, voteVariables>({
            mutation,
            input: { isUpvote, voteFor, forId: objectId },
            onSuccess: () => { onChange(isUpvote) },
        })
    }, [objectId, voteFor, onChange, mutation]);

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
        const upvoteColor = (!userId || disabled) ? "rgb(189 189 189)" :
            internalIsUpvoted === true ? "#34c38b" :
                "#687074";
        const UpvoteIcon = direction === "column" ? UpvoteWideIcon : UpvoteTallIcon;
        return { UpvoteIcon, upvoteColor };
    }, [userId, disabled, internalIsUpvoted, direction]);

    const { DownvoteIcon, downvoteColor } = useMemo(() => {
        const downvoteColor = (!userId || disabled) ? "rgb(189 189 189)" :
            internalIsUpvoted === false ? "#af2929" :
                "#687074";
        const DownvoteIcon = direction === "column" ? DownvoteWideIcon : DownvoteTallIcon;
        return { DownvoteIcon, downvoteColor };
    }, [userId, disabled, internalIsUpvoted, direction]);

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