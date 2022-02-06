// Used to display popular/search results of a particular object type
import { Box, CircularProgress, Link, List, Stack, Tooltip, Typography } from '@mui/material';
import { FeedListProps } from '../types';
import { clickSize, containerShadow } from 'styles';
import { useCallback } from 'react';

export function FeedList({
    title = 'Popular Items',
    onClick,
    loading,
    children,
}: FeedListProps) {
    const handleSeeMoreClick = useCallback(() => { onClick() }, [onClick]);

    return (
        <Box
            onClick={handleSeeMoreClick}
            sx={{
                ...containerShadow,
                borderRadius: '8px',
                background: (t) => t.palette.background.default,
                transition: 'scale filter 1s scale 1s ease-in-out',
                cursor: 'pointer',
                '&:hover': {
                    transform: 'scale(1.02)',
                    filter: `brightness(105%)`,
                },
            }}
        >
            <Box sx={{
                background: (t) => t.palette.primary.dark,
                color: (t) => t.palette.primary.contrastText,
                borderRadius: '8px 8px 0 0',
                padding: 0.5,
            }}>
                <Typography component="h2" variant="h4" textAlign="center">{title}</Typography>
            </Box>
            <Tooltip placement="bottom" title="Press to see more">
                <Stack direction="column">
                    <Box sx={{ minHeight: 'min(300px, 25vh)' }}>
                        {loading ?
                            (
                                <List sx={{ minHeight: 'min(300px, 25vh)' }}>
                                    {children}
                                </List>
                            ) : (
                                <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                                    <CircularProgress color="secondary" />
                                </Box>
                            )
                        }
                    </Box>
                    <Link onClick={handleSeeMoreClick}>
                        <Typography
                            sx={{
                                ...clickSize,
                                color: (t) => t.palette.secondary.dark,
                                display: 'flex',
                                alignItems: 'center',
                                flexDirection: 'row-reverse',
                                marginRight: 2,
                                marginBottom: 1,
                            }}
                        >
                            See more results
                        </Typography>
                    </Link>
                </Stack>
            </Tooltip>
        </Box>
    )
}