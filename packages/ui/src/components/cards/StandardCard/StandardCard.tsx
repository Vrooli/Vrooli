import { makeStyles } from '@mui/styles';
import { Card, CardActionArea, CardActions, CardContent, IconButton, Theme, Tooltip, Typography } from '@mui/material';
import { StandardCardProps } from '../types';
import { cardStyles } from '../styles';
import { combineStyles } from 'utils';
import { Launch as LaunchIcon } from '@mui/icons-material';
import { useCallback } from 'react';

const componentStyles = (theme: Theme) => ({
    
});

const useStyles = makeStyles(combineStyles(cardStyles, componentStyles));

export const StandardCard = ({
    data,
    onClick = () => {},
}: StandardCardProps) => {
    const classes = useStyles();

    const handleClick = useCallback(() => onClick(data.name), [data, onClick]);

    return (
        <Card className={classes.cardRoot} onClick={handleClick}>
        <CardActionArea>
            <CardContent className={`${classes.content}`}>
                <Typography gutterBottom variant="h6" component="h3">
                    {data.name}
                </Typography>
            </CardContent>
        </CardActionArea>
        <CardActions>
            <Tooltip title="View" placement="bottom">
                <IconButton onClick={handleClick}>
                    <LaunchIcon className={classes.icon} />
                </IconButton>
            </Tooltip>
        </CardActions>
    </Card>
    )
}