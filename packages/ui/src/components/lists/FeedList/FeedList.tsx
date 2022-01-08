// Used to display popular/search results of a particular object type
import { Box, Tooltip, Typography } from '@mui/material';
import { FeedListProps } from '../types';
import { centeredText, containerShadow } from 'styles';
import { useCallback, useMemo } from 'react';

export function FeedList<DataType>({
    title = 'Popular Items',
    data,
    cardFactory,
    onClick,
}: FeedListProps<DataType>) {
    const cards = useMemo(() => data ? ((Object.values(data) as any)?.edges?.map((edge, index) => cardFactory(edge.node, index))) : null, [cardFactory, data]);

    const handleContainerClick = useCallback(() => onClick(), [onClick]);

    return (
        <Box
            onClick={handleContainerClick}
            sx={{
                transition: 'filter 1s scale 1s ease-in-out',
                '&:hover': {
                    transform: 'scale(1.05)',
                    filter: `brightness(105%)`,
                },
            }}
        >
            <Typography component="h2" variant="h4" sx={{ ...centeredText }}>{title}</Typography>
            <Tooltip placement="bottom" title="Press to see more">
                <Box
                    sx={{
                        ...containerShadow,
                        borderRadius: '16px',
                        background: (t) => t.palette.background.default,
                        minHeight: 'min(300px, 25vh)'
                    }}
                >
                    {cards}
                </Box>
            </Tooltip>
        </Box>
    )
}