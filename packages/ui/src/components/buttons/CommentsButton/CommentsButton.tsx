import { Box, ListItemText, Stack, Tooltip, useTheme } from '@mui/material';
import { useCallback } from 'react';
import { CommentsButtonProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useLocation } from '@shared/route';
import { getObjectSlug, getObjectUrlBase } from 'utils';
import { CommentIcon } from '@shared/icons';

export const CommentsButton = ({
    commentsCount = 0,
    disabled = false,
    object,
    tooltipPlacement = "left",
}: CommentsButtonProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    // When clicked, navigate to object's comment section
    const handleClick = useCallback((e: any) => {
        // Stop propagation
        e.stopPropagation();
        if (!object) return;
        setLocation(`${getObjectUrlBase(object)}/${getObjectSlug(object)}#comments`)
    }, [object, setLocation]);

    return (
        <Stack
            direction="row"
            spacing={0.5}
            sx={{
                marginRight: 0,
                pointerEvents: 'none',
            }}
        >
            <Tooltip placement={tooltipPlacement} title={'View comments'}>
                <Box onClick={handleClick} sx={{
                    display: 'contents',
                    cursor: disabled ? 'none' : 'pointer',
                    pointerEvents: disabled ? 'none' : 'all',
                }}>
                    <CommentIcon fill={disabled ? 'rgb(189 189 189)' : palette.secondary.main} />
                </Box>
            </Tooltip>
            <ListItemText
                primary={commentsCount}
                sx={{ ...multiLineEllipsis(1), pointerEvents: 'none' }}
            />
        </Stack>
    )
}