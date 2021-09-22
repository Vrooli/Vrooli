import React from 'react';
import {
    Box,
    Button,
    Typography,
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { combineStyles, LINKS } from 'utils';
import { slideStyles } from './styles';
import { useHistory } from 'react-router';

const componentStyles = (theme) => ({

})

const useStyles = makeStyles(combineStyles(slideStyles, componentStyles));

function Hero({
    image,
    ...props
}) {
    const classes = useStyles();
    const history = useHistory();

    return (
        <Box className={classes.slideRoot}>
            <img className={classes.slideBackground} alt={image?.url} src={image?.src} />
            <div className={classes.slidePad}>
                <Typography variant='h2' component='h1' className={`${classes.titleCenter} ${classes.textPop}`}>Your portal to idea monetization</Typography>
                <Button
                    className={classes.buttonCenter}
                    type="submit"
                    color="secondary"
                    onClick={() => history.push(LINKS.Mission)}
                >
                    Learn More
                </Button>
            </div>
        </Box>
    );
}

export { Hero };