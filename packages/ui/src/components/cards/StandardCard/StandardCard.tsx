import { Card, CardActionArea, CardContent, Theme, Typography } from '@mui/material';
import { StandardCardProps } from '../types';
import { cardContent, cardRoot } from '../styles';
import { useCallback } from 'react';

export const StandardCard = ({
    data,
    onClick = () => { },
}: StandardCardProps) => {
    const handleClick = useCallback(() => data.id && onClick(data.id), [data, onClick]);

    return (
        <Card onClick={handleClick} sx={{ ...cardRoot }}>
            <CardActionArea>
                <CardContent sx={{ ...cardContent }}>
                    <Typography gutterBottom variant="h6" component="h3">
                        {data.name}
                    </Typography>
                </CardContent>
            </CardActionArea>
        </Card>
    )
}