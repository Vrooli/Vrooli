import {
    Card,
    CardActionArea,
    CardContent,
    CardMedia,
    Tooltip,
    Typography
} from '@mui/material';
import { openLink } from 'utils';
import { NoImageWithTextIcon } from 'assets/img';
import { useEffect, useMemo } from 'react';
import { readOpenGraphQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { useLocation } from 'wouter';
import { ResourceCardProps } from '../types';
import { cardContent, cardRoot } from '../styles';
import { multiLineEllipsis } from 'styles';
import { readOpenGraph, readOpenGraphVariables } from 'graphql/generated/readOpenGraph';

export const ResourceCard = ({
    data
}: ResourceCardProps) => {
    const [, setLocation] = useLocation();
    const [getOpenGraphData, { data: queryResult }] = useLazyQuery<readOpenGraph, readOpenGraphVariables>(readOpenGraphQuery);
    const queryData = useMemo(() => queryResult?.readOpenGraph, [queryResult]);
    const title = useMemo(() => data?.title ?? queryData?.title, [data, queryData]);
    const url = useMemo(() => data?.link ?? queryData?.site, [data, queryData]);

    useEffect(() => {
        if (data.link) {
            getOpenGraphData({ variables: { input: { url: data.link } } })
        }
    }, [getOpenGraphData, data])

    const display = useMemo(() => {
        if (!data || !data.displayUrl) {
            return <NoImageWithTextIcon style={{ aspectRatio: '1' }} />
        }
        return (
            <CardMedia
                component="img"
                src={data.displayUrl}
                sx={{ aspectRatio: '1' }}
                alt={`Image from ${data.link ?? data.title}`}
                title={data.title ?? data.link}
            />
        )
    }, [data])

    return (
        <Tooltip placement="top" title={data?.description ?? queryData?.description ?? ''}>
            <Card 
                onClick={() => openLink(setLocation, url)}
                sx={{
                    ...cardRoot,
                    width: 'max(50px, 15vh)',
                    height: '100%',
                }}
            >
                <CardActionArea>
                    {display}
                    <CardContent sx={{...cardContent}}>
                        <Typography
                            gutterBottom
                            variant="body1"
                            component="h3"
                            sx={{...multiLineEllipsis(2)}}
                        >
                            {title}
                        </Typography>
                    </CardContent>
                </CardActionArea>
            </Card>
        </Tooltip>
    )
}