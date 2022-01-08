// Used to display popular/search results of a particular object type
import { Box, Typography } from '@mui/material';
import { FeedListProps } from '../types';
import { centeredText } from 'styles';
import { useMemo } from 'react';

export function FeedList<DataType>({
    title = 'Popular Items',
    data,
    cardFactory,
}: FeedListProps<DataType>) {
    const cards = useMemo(() => data ? ((Object.values(data) as any)?.edges?.map((edge, index) => cardFactory(edge.node, index))) : null, [cardFactory, data]);

    return (
        <Box sx={{borderRadius: '25%', background: 'background.paper', minHeight: 'min(300px, 25vh)'}}>
            <Typography component="h2" variant="h4" sx={{ ...centeredText }}>{title}</Typography>
            <div>
                {cards}
            </div>
        </Box>
    )
}