import {
    Card,
    CardContent,
    Typography
} from '@mui/material';
import { BarGraph, Dimensions } from 'components';
import { useEffect, useRef, useState } from 'react';
import { cardRoot } from '../styles';
import { StatCardProps } from '../types';

export const StatCard = ({
    title = 'Daily active users',
    data,
    index,
}: StatCardProps) => {
    const [dimensions, setDimensions] = useState<Dimensions>({ width: undefined, height: undefined });
    const ref = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const width = ref.current?.clientWidth ? ref.current.clientWidth - 50 : 0;
        const height = ref.current?.clientHeight ? ref.current.clientHeight - 50 : 0;
        setDimensions({ width, height });
    }, [setDimensions])

    return (
        <Card ref={ref} sx={{ 
            ...cardRoot,
            width: '100%',
            height: '100%', 
        }}>
            <CardContent
                sx={{
                    display: 'contents',
                }}
            >
                <Typography
                    variant="h6"
                    component="h2"
                    textAlign="center"
                    sx={{
                        marginBottom: '10px'
                    }}
                >{title}</Typography>
                <BarGraph
                    dimensions={dimensions}
                    style={{
                        width: '90%',
                        height: '85%',
                        display: 'block',
                        margin: 'auto',
                    }}
                />
            </CardContent>
        </Card>
    );
}