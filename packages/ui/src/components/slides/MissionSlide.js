import React from 'react';
import {
    Box,
    Button,
    Container,
    Grid,
    Typography,
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { combineStyles, LINKS } from 'utils';
import { slideStyles } from './styles';
import { useHistory } from 'react-router';

const componentStyles = (theme) => ({

})

const useStyles = makeStyles(combineStyles(slideStyles, componentStyles));

function MissionSlide({
    ...props
}) {
    const classes = useStyles();
    const history = useHistory();

    return (
        <Box className={classes.slideRoot}>
            <div className={classes.slidePad}>
                <Typography variant='h2' component='h1' className={classes.titleCenter}>Our Mission</Typography>
                <Grid container spacing={1}>
                    <Grid item xs={6}>
                        <Typography variant='h3' component='h2'>Bring the power of entrepreneurship to the masses, through the use of visual work routines</Typography>
                    </Grid>
                    <Grid item xs={6}>
                    </Grid>
                </Grid>
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

export { MissionSlide };