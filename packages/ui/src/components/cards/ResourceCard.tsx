import {
    Card,
    CardActionArea,
    CardContent,
    CardMedia,
    Theme,
    Tooltip,
    Typography
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { combineStyles, openLink } from 'utils';
import { NoImageWithTextIcon } from 'assets/img';
import { cardStyles } from './styles';
import { useEffect, useMemo } from 'react';
import { Resource } from 'types';
import { readOpenGraphQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { useNavigate } from 'react-router-dom';

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

interface Props {
    resource: Resource
}

export const ResourceCard = ({
    resource
}: Props) => {
    const classes = useStyles();
    const navigate = useNavigate();
    const [getOpenGraphData, { data: queryResult }] = useLazyQuery<any, any>(readOpenGraphQuery);
    const data = useMemo(() => queryResult?.readOpenGraph, [queryResult]);
    const title = useMemo(() => resource?.title ?? data?.title, [resource, data]);
    const url = useMemo(() => resource?.url ?? data?.url, [resource, data]);

    useEffect(() => {
        if (resource.url) {
            getOpenGraphData({ variables: { input: { url: resource.url } } })
        }
    }, [getOpenGraphData, resource])

    const display = useMemo(() => {
        if (!data) {
            return <NoImageWithTextIcon className={classes.displayImage} />
        }
        return (
            <CardMedia
                component="img"
                src={data.imageUrl}
                className={classes.displayImage}
                alt={`Image from ${data.site ?? data.title}`}
                title={data.title ?? data.site}
            />
        )
    }, [classes.displayImage, data])

    return (
        <Tooltip placement="top" title={resource?.description ?? data?.description}>
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