// Used to display popular/search results of a particular object type
import { Box, Link, List, Tooltip, Typography } from '@mui/material';
import { FeedListProps } from '../types';
import { clickSize, containerShadow } from 'styles';
import { useCallback } from 'react';

export function FeedList({
    title = 'Popular Items',
    onClick,
    children,
}: FeedListProps) {
    const handleContainerClick = useCallback(() => onClick(), [onClick]);

    return (
        <Box
            onClick={handleContainerClick}
            sx={{
                transition: 'filter 1s scale 1s ease-in-out',
                cursor: 'pointer',
                '&:hover': {
                    transform: 'scale(1.05)',
                    filter: `brightness(105%)`,
                },
            }}
        >
            <Typography component="h2" variant="h4" textAlign="center">{title}</Typography>
            <Tooltip placement="bottom" title="Press to see more">
                <Box
                    sx={{
                        ...containerShadow,
                        borderRadius: '8px',
                        background: (t) => t.palette.background.default,
                        minHeight: 'min(300px, 25vh)'
                    }}
                >
                    <List>
                        {children}
                    </List>
                    <Link onClick={handleContainerClick}>
                        <Typography
                            sx={{
                                ...clickSize,
                                color: (t) => t.palette.secondary.dark,
                                display: 'flex',
                                alignItems: 'center',
                                flexDirection: 'row-reverse',
                                marginRight: 1,
                            }}
                        >
                            See more results
                        </Typography>
                    </Link>
                </Box>
            </Tooltip>
        </Box>
    )
}