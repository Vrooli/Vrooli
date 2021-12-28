import { makeStyles } from '@mui/styles';
import { Card, CardActionArea, CardActions, CardContent, IconButton, Theme, Tooltip, Typography } from '@mui/material';
import { OrganizationCardProps } from '../types';
import { cardStyles } from '../styles';
import { combineStyles } from 'utils';
import { Launch as LaunchIcon } from '@mui/icons-material';

const componentStyles = (theme: Theme) => ({
    
});

const useStyles = makeStyles(combineStyles(cardStyles, componentStyles));

export const OrganizationCard = ({
    name = 'Default name',
    onClick = () => {},
}: OrganizationCardProps) => {
    const classes = useStyles();

    return (
        <Card className={classes.cardRoot} onClick={() => {}}>
        <CardActionArea>
            <CardContent className={`${classes.content}`}>
                <Typography gutterBottom variant="h6" component="h3">
                    {name}
                </Typography>
            </CardContent>
        </CardActionArea>
        <CardActions>
            <Tooltip title="View" placement="bottom">
                <IconButton onClick={() => {}}>
                    <LaunchIcon className={classes.icon} />
                </IconButton>
            </Tooltip>
        </CardActions>
    </Card>
    )
}