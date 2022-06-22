import { ListItemText, Stack, Tooltip } from '@mui/material';
import { useCallback } from 'react';
import { CommentButtonProps } from '../types';
import { Forum as CommentsIcon } from '@mui/icons-material';
import { multiLineEllipsis } from 'styles';

export const CommentButton = ({
    commentsCount = 0,
    objectId,
    objectType,
    tooltipPlacement = "left",
}: CommentButtonProps) => {

    // When clicked, navigate to object's comment section
    const handleClick = useCallback(() => {
        //TODO
    }, []);

    return (
        <Stack
            direction="row"
            spacing={1}
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