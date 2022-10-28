import { Box, ListItemText, Stack, useTheme } from '@mui/material';
import { useCallback, useMemo } from 'react';
import { CommentsButtonProps } from '../types';
import { multiLineEllipsis } from 'styles';
import { useLocation } from '@shared/route';
import { CommentIcon } from '@shared/icons';
import { getObjectUrl } from 'utils';

export const CommentsButton = ({
    commentsCount = 0,
    disabled = false,
    object,
}: CommentsButtonProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    // When clicked, navigate to object's comment section
    const link = useMemo(() => object ? `${getObjectUrl(object)}#comments` : '', [object]);
    const handleClick = useCallback((event: any) => {
        // Stop propagation to prevent list item from being selected
        event.stopPropagation();
        // Prevent default to stop href from being used
        event.preventDefault();
        if (link.length === 0) return;
        setLocation(link);
    }, [link, setLocation]);

    return (
        <Stack
            direction="row"
            spacing={0.5}
            sx={{
                marginRight: 0,
                pointerEvents: 'none',
            }}
        >
            <Box
                component="a"
                href={link}
                onClick={handleClick}
                sx={{
                    display: 'contents',
                    cursor: disabled ? 'none' : 'pointer',
                    pointerEvents: disabled ? 'none' : 'all',
                }}>
                <CommentIcon fill={disabled ? 'rgb(189 189 189)' : palette.secondary.main} />
            </Box>
            <ListItemText
                primary={commentsCount}
                sx={{ ...multiLineEllipsis(1), pointerEvents: 'none' }}
            />
        </Stack>
    )
}