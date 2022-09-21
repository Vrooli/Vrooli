import { ListItemText, Stack, Tooltip } from '@mui/material';
import { useCallback } from 'react';
import { CommentsButtonProps } from '../types';
import { Forum as CommentsIcon } from '@mui/icons-material';
import { multiLineEllipsis } from 'styles';
import { useLocation } from '@shared/route';
import { getObjectSlug, getObjectUrlBase } from 'utils';

export const CommentsButton = ({
    commentsCount = 0,
    object,
    tooltipPlacement = "left",
}: CommentsButtonProps) => {
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
            }}
        >
            <Tooltip placement={tooltipPlacement} title={'View comments'}>
                <CommentsIcon
                    onClick={handleClick}
                    sx={{
                        fill: '#76a7c3',
                        cursor: 'pointer'
                    }}
                />
            </Tooltip>
            <ListItemText
                primary={commentsCount}
                sx={{ ...multiLineEllipsis(1) }}
            />
        </Stack>
    )
}