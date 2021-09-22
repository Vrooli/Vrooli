import React, { useCallback } from 'react';
import {
    Box,
    Button,
    Grid,
    Typography,
} from '@material-ui/core';
import { useHistory } from 'react-router';
import { YoutubeEmbed } from 'components';
import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles((theme) => ({
    slideRoot: {
        display: 'flex',
        overflowX: 'hidden',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
    },
    slideBackground: {
        position: 'absolute',
        width: '100vw',
        height: '100vh',
        objectFit: 'cover',
    },
    slidePad: {
        padding: theme.spacing(2),
        zIndex: 1,
    },
    titleCenter: {
        textAlign: 'center',
        paddingLeft: '5vw',
        paddingRight: '5vw',
        paddingBottom: '1em',
    },
    mainGrid: {
        paddingBottom: '2em',
    },
    buttonCenter: {
        display: 'flex',
        margin: 'auto',
    },
    textPop: {
        padding: '0',
        color: 'white',
        textAlign: 'center',
        fontWeight: '600',
        textShadow:
            `-1px -1px 0 black,  
                1px -1px 0 black,
                -1px 1px 0 black,
                1px 1px 0 black`
    },
    [theme.breakpoints.down(500)]: {
        slideRoot: {
            minHeight: '50vh',
        },
        slideBackground: {
            height: '50vh',
        }
    },
}))

function Slide({
    data,
    width
}) {
    const classes = useStyles();
    const history = useHistory();

    // Background can either be a solid color or an image
    const toBackground = (background) => {
        console.log('in to background', background?.image?.src)
        if (!background) return null;
        if (background.image) {
            return <img className={classes.slideBackground} alt={background?.image?.alt} src={background?.image?.src} />
        }
        // TODO solid color option
        return null;
    }

    const toTitle = (title) => {
        if (!title) return null;
        let titleClasses = `${classes.titleCenter}`;
        if (title.style === 'pop') titleClasses += ` ${classes.textPop}`;
        return <Typography variant='h2' component='h1' className={titleClasses}>{title.text}</Typography>
    }

    const toBodyChild = useCallback((child) => {
        console.log('in body child', child)
        if (!child) return null;
        // If child has 'content' property, then it is a GridItem
        if (Array.isArray(child.content)) {
            return <Grid item xs={child.xs ?? 4}>
                {child.content.map(grandchild => toBodyChild(grandchild))}
            </Grid>
        }
        console.log('here', child)
        // If child has 'title' property, then it is a Typography
        if (child.title) return (
            <Typography 
                variant={child.title.variant ?? 'h4'} 
                component={child.title.component ?? 'h2'} 
                textAlign={child.title.textAlign ?? 'left'}
            >
                {child.title.text}
            </Typography>
        )
        // If child has 'video' property, then it is a YoutubeEmbed
        if (child.video) return <YoutubeEmbed embedId={child.video.link} width={Math.min(width-16, 1000)} height={Math.floor(Math.min(width-16, 1000)*9/16)} />
        //TODO
        if (child.image) return null;
        //TODO
        return null;
    }, [width])

    const toBody = (body) => {
        console.log('in body', body)
        if (!Array.isArray(body)) return null;
        return <Grid className={classes.mainGrid} container spacing={1}>
            {body.map(child => toBodyChild(child))}
        </Grid>
    }

    const toButton = (button) => {
        if (!button) return null;
        let buttonClasses = `${classes.buttonCenter}`;
        return <Button
            className={buttonClasses}
            type="submit"
            size="large"
            color={button.color ?? 'secondary'}
            onClick={() => history.push(button.link)}
        >
            {button.text}
        </Button>
    }

    return (
        <Box className={classes.slideRoot}>
            {toBackground(data?.background)}
            <div className={classes.slidePad}>
                {toTitle(data?.title)}
                {toBody(data?.body)}
                {toButton(data?.button)}
            </div>
        </Box>
    );
}

export { Slide };