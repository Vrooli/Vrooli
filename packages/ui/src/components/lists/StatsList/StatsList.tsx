import { Box } from '@mui/material';
import { StatCard } from 'components';
import { useMemo } from 'react';
import { StatsListProps } from '../types';

export const StatsList = ({
    data,
}: StatsListProps) => {
    const statCards = useMemo(() => (
        data?.map((item, index) => (
            <Box
                sx={{
                    margin: 2,
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}
            >
                <StatCard
                    key={index}
                    index={index}
                    data={item}
                />
            </Box>
        ))
    ), [data]);

    return (
        <Box
            sx={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))',
                gridAutoRows: '350px',
                alignItems: 'stretch',
            }}
        >
            {statCards}
        </Box>
    )
}