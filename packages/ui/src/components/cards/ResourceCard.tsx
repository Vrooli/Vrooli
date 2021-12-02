import {
    Card,
    CardActionArea,
    CardActions,
    CardContent,
    CardMedia,
    IconButton,
    Theme,
    Tooltip,
    Typography
} from '@material-ui/core';
import { Launch as LaunchIcon } from '@material-ui/icons';
import { makeStyles } from '@material-ui/styles';
import { combineStyles } from 'utils';
import { NoImageWithTextIcon } from 'assets/img';
import { cardStyles } from './styles';
import { useEffect, useMemo, useState } from 'react';
import { Resource } from 'types';
import og from 'open-graph';

const componentStyles = (theme: Theme) => ({
    root: {
        // width: 'max(50px, 20vh)',
        // display: 'inline-table',
        border: '2px solid green',
        height: '100%',
        width: '100%',
    }
});

const useStyles = makeStyles(combineStyles(cardStyles, componentStyles));

interface Props {
    resource: Resource
}

export const ResourceCard = ({
    resource
}: Props) => {
    const classes = useStyles();
    const [openGraphData, setOpenGraphData] = useState<any>(undefined)

    useEffect(() => {
        if (resource.url) {
            console.log('Getting OpenGraph data...')
            og(resource.url, (err: any, meta: any) => {
                console.log('GOT OPEN GRAPH DATA', meta);
                if (err) console.error('ERROR GETTING OPEN GRAPH DATA', err);
                setOpenGraphData(meta);
            })
        }
    }, [resource])

    const display = useMemo(() => openGraphData?.image?.url ? (
        <CardMedia 
            component="img" 
            src={openGraphData.image.url} 
            className={classes.displayImage} 
            alt={`Image from ${openGraphData.site_name ?? openGraphData.title}`} 
            title={openGraphData.title} 
        />
    ) : (
        <NoImageWithTextIcon className={classes.displayImage} />
    ), [classes.displayImage, openGraphData])

    return (
        <Card className={`${classes.root} ${classes.cardRoot}`}>
            <CardActionArea>
                {display}
                <CardContent className={`${classes.content} ${classes.topMargin}`}>
                    <Typography gutterBottom variant="h6" component="h3">
                        {resource.title}
                    </Typography>
                </CardContent>
            </CardActionArea>
            <CardActions>
                <Tooltip title="View" placement="bottom">
                    <IconButton onClick={() => { }}>
                        <LaunchIcon className={classes.icon} />
                    </IconButton>
                </Tooltip>
            </CardActions>
        </Card>
    )
}