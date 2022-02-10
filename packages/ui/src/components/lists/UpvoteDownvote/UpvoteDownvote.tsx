import { useMutation } from '@apollo/client';
import { Box, Stack, Tooltip, Typography } from '@mui/material';
import { vote } from 'graphql/generated/vote';
import { voteMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { UpvoteDownvoteProps } from '../types';

export const UpvoteDownvote = ({
    session,
    score,
    isUpvoted,
    objectId,
    voteFor,
    onChange,
}: UpvoteDownvoteProps) => {
    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsUpvoted, setInternalIsUpvoted] = useState<boolean | null>(isUpvoted ?? null);
    useEffect(() => setInternalIsUpvoted(isUpvoted ?? null), [isUpvoted]);

    const internalScore = useMemo(() => {
        console.log('internalScore start', score);
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

    const [mutation] = useMutation<vote>(voteMutation);
    const handleVote = useCallback((e: any, isUpvote: boolean | null) => {
        // Prevent propagation of normal click event
        e.stopPropagation();
        // Send vote mutation
        mutationWrapper({
            mutation,
            input: { isUpvote, voteFor, forId: objectId },
            onSuccess: (response) => { onChange(response.data.vote) },
        })
    }, [objectId, voteFor]);

    const handleUpvoteClick = useCallback((event: any) => {
        if (!session.id) return;
        // If already upvoted, cancel the vote
        const vote = internalIsUpvoted === true ? null : true;
        setInternalIsUpvoted(vote);
        handleVote(event, vote);
    }, [session.id, internalIsUpvoted, handleVote]);

    const handleDownvoteClick = useCallback((event: any) => {
        if (!session.id) return;
        // If already downvoted, cancel the vote
        const vote = internalIsUpvoted === false ? null : false;
        setInternalIsUpvoted(vote);
        handleVote(event, vote);
    }, [session.id, internalIsUpvoted, handleVote]);

    const upvoteColor = useMemo(() => {
        if (!session.id) return "rgb(189 189 189)";
        if (internalIsUpvoted === true) return "#34c38b";
        return "#687074";
    }, [internalIsUpvoted, session.id]);

    const downvoteColor = useMemo(() => {
        if (!session.id) return "rgb(189 189 189)";
        if (internalIsUpvoted === false) return "#af2929";
        return "#687074";
    }, [internalIsUpvoted, session.id]);

    return (
        <Stack direction="column">
            {/* Upvote arrow */}
            <Tooltip title="Upvote" placement="left" >
                <Box
                    display="inline-block"
                    onClick={handleUpvoteClick}
                    sx={{
                        cursor: session.id ? 'pointer' : 'default',
                        '&:hover': {
                            filter: session.id ? `brightness(120%)` : 'none',
                            transition: 'filter 0.2s',
                        },
                    }}
                >
                    <svg width="36" height="36">
                        <path d="M2 26h32L18 10 2 26z" fill={upvoteColor}></path>
                    </svg>
                </Box>
            </Tooltip>
            {/* Score */}
            <Typography variant="body1" textAlign="center">{internalScore}</Typography>
            {/* Downvote arrow */}
            <Tooltip title="Downvote" placement="left">
                <Box
                    display="inline-block"
                    onClick={handleDownvoteClick}
                    sx={{
                        cursor: session.id ? 'pointer' : 'default',
                        '&:hover': {
                            filter: session.id ? `brightness(120%)` : 'none',
                            transition: 'filter 0.2s',
                        },
                    }}
                >
                    <svg width="36" height="36">
                        <path d="M2 10h32L18 26 2 10z" fill={downvoteColor}></path>
                    </svg>
                </Box>
            </Tooltip>
        </Stack>
    )
}