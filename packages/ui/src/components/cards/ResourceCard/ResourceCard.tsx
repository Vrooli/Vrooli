import {
    Card,
    CardActionArea,
    CardContent,
    CardMedia,
    Theme,
    Tooltip,
    Typography
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import { combineStyles, openLink } from 'utils';
import { NoImageWithTextIcon } from 'assets/img';
import { cardStyles } from '../styles';
import { useEffect, useMemo } from 'react';
import { readOpenGraphQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { useNavigate } from 'react-router-dom';
import { ResourceCardProps } from '../types';
import { readOpenGraph, readOpenGraphVariables } from 'graphql/generated/readOpenGraph';

const componentStyles = (theme: Theme) => ({
    root: {
        width: 'max(50px, 15vh)',
        // display: 'inline-table',
        border: '2px solid green',
        height: '100%',
        //width: '100%',
    },
    displayImage: {
        border: '2px solid pink',
        aspectRatio: '1',
    },
    multiLineEllipsis: {
        display: '-webkit-box',
        '-webkit-line-clamp': 2,
        '-webkit-box-orient': 'vertical',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
    },
});

const useStyles = makeStyles(combineStyles(cardStyles, componentStyles));

export const ResourceCard = ({
    data
}: ResourceCardProps) => {
    const classes = useStyles();
    const navigate = useNavigate();
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
            return <NoImageWithTextIcon className={classes.displayImage} />
        }
        return (
            <CardMedia
                component="img"
                src={data.displayUrl}
                className={classes.displayImage}
                alt={`Image from ${data.link ?? data.title}`}
                title={data.title ?? data.link}
            />
        )
    }, [classes.displayImage, data])

    return (
        <Tooltip placement="top" title={data?.description ?? queryData?.description ?? ''}>
            <Card className={`${classes.root} ${classes.cardRoot}`} onClick={() => openLink(navigate, url)}>
                <CardActionArea>
                    {display}
                    <CardContent className={`${classes.content} ${classes.topMargin}`}>
                        <Typography
                            className={classes.multiLineEllipsis}
                            gutterBottom
                            variant="body1"
                            component="h3"
                        >
                            {title}
                        </Typography>
                    </CardContent>
                </CardActionArea>
            </Card>
        </Tooltip>
    )
}