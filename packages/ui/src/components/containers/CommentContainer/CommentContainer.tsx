/**
 * Contains new comment input, and list of Reddit-style comments.
 */
import { Box, CircularProgress, Link, Stack, Typography, useTheme } from '@mui/material';
import { CommentContainerProps } from '../types';
import { containerShadow } from 'styles';

export function CommentContainer({
    objectID,
    objectType,
    onCommentSubmit,
}: CommentContainerProps) {
    const { palette } = useTheme();

    return (
        <>
            <Typography component="h2" variant="h5" textAlign="left">Comments</Typography>
            <Box
                sx={{
                    ...containerShadow,
                    borderRadius: '8px',
                    overflow: 'overlay',
                    background: palette.background.default,
                    width: 'min(100%, 700px)',
                }}
            >
                {/* Add comment */}
                <Box>

                </Box>
                {/* Comments list */}
                <Stack direction="column">

                </Stack>
            </Box>
        </>
    );
}