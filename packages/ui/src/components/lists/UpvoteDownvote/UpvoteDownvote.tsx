import { Box, Stack, Typography } from '@mui/material';
import { useCallback, useEffect, useState } from 'react';
import { UpvoteDownvoteProps } from '../types';

export const UpvoteDownvote = ({
    votes,
    isUpvoted,
    onVote,
}: UpvoteDownvoteProps) => {
    // Used to respond to user clicks immediately, without having 
    // to wait for the mutation to complete
    const [internalIsUpvoted, setInternalIsUpvoted] = useState<boolean | null>(isUpvoted ?? null);
    useEffect(() => setInternalIsUpvoted(isUpvoted ?? null), [isUpvoted]);

    const handleUpvoteClick = useCallback((event: any) => {
        // If already upvoted, cancel the vote
        const vote = internalIsUpvoted === true ? null : true;
        setInternalIsUpvoted(vote);
        onVote(event, vote);
    }, [internalIsUpvoted, onVote]);

    const handleDownvoteClick = useCallback((event: any) => {
        // If already downvoted, cancel the vote
        const vote = internalIsUpvoted === false ? null : false;
        setInternalIsUpvoted(vote);
        onVote(event, vote);
    }, [internalIsUpvoted, onVote]);

    return (
        <Stack direction="column">
            {/* Upvote arrow */}
            <Box display="inline-block" sx={{ cursor: 'pointer' }} onClick={handleUpvoteClick}>
                <svg width="36" height="36">
                    <path d="M2 10h32L18 26 2 10z" fill={isUpvoted ? '#f48024' : '#687074'}></path>
                </svg>
            </Box>
            {/* Score */}
            <Typography variant="body2">{votes}</Typography>
            {/* Downvote arrow */}
            <Box display="inline-block" sx={{ cursor: 'pointer' }} onClick={handleDownvoteClick}>
                <svg width="36" height="36">
                    <path d="M2 10h32L18 26 2 10z" fill={!isUpvoted ? '#f48024' : '#687074'}></path>
                </svg>
            </Box>
        </Stack>
    )
}