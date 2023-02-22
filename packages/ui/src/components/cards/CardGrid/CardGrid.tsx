import { Box } from '@mui/material';
import { CardGridProps } from '../types';

export const CardGrid = ({
    children,
    minWidth,
}: CardGridProps) => {
    return (
        <Box sx={{
            display: 'grid',
            gridTemplateColumns: `repeat(auto-fit, minmax(${minWidth}px, 1fr))`,
            alignItems: 'stretch',
            gap: 2,
            margin: 2,
            borderRadius: 2,
        }}>
            {children}
        </Box>
    );
}