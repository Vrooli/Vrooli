import React from 'react';
import { InformationalBreadcrumbs } from 'components';
import {
    Grid,
    Typography
} from '@material-ui/core';
import Facebook from 'assets/img/Facebook.png';
import Instagram from 'assets/img/Instagram.png';
import { makeStyles } from '@material-ui/styles';
import { useTheme } from '@emotion/react';
import { pageStyles } from './styles';
import { combineStyles } from 'utils';

const componentStyles = (theme) => ({
    social: {
        width: '80px',
        height: '80px',
    }
})

const useStyles = makeStyles(combineStyles(pageStyles, componentStyles));

function AboutPage({
    business,
}) {
    const classes = useStyles();
    const theme = useTheme();
    const socials = [
        [Facebook, 'Facebook', business?.SOCIAL?.Facebook],
        [Instagram, 'Instagram', business?.SOCIAL?.Instagram],
    ]

    return (
        <div id='page'>
            <InformationalBreadcrumbs textColor={theme.palette.secondary.dark} />
            <br/>
            <Grid container spacing={2}>
                <Grid item md={12} lg={8}>
                    <div className={classes.header}>
                        <Typography variant="h3" component="h1">Our Story</Typography>
                    </div>
                    <p>For 2000 Urras years, Far Out Rides has been striving to Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>
                    <p>Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.</p>
                    <p>Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo. Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur and contact us if you would like more information, or to speak with one of our experts at <a href={business?.PHONE?.Link}>{business?.PHONE?.Label}</a>.</p>
                    <p>Happy exploring,</p>
                    <h2 style={{ fontFamily: `'Lato', sans-serif;` }}>The Squanch Brothers</h2>
                </Grid>
                <Grid item md={12} lg={4}>
                    <div className={classes.header}>
                        <Typography variant="h4" component="h2">Check out our socials</Typography>
                    </div>
                    {socials.map(s => (
                        <a href={s[2]} target="_blank" rel="noopener noreferrer">
                            <img className={classes.social} alt={s[1]} src={s[0]} />
                        </a>
                    ))}
                </Grid>
            </Grid>
        </div >
    );
}

AboutPage.propTypes = {
}

export { AboutPage };