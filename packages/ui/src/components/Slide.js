import React, { useCallback, useMemo } from 'react';
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
        width: '100%',
        height: '100%',
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
    bodyGridItem: {
        paddingBottom: '1em',
    },
    bodyList: {
        listStyleType: 'circle',
        fontSize: 'xxx-large',
    },
    bodyImageContainer: {
        justifyContent: 'center',
        height: '100%',
        display: 'flex',
    },
    bodyImage: {
        maxWidth: '90%',
        maxHeight: '100%',
        objectFit: 'contain',
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
    },
    [theme.breakpoints.down('sm')]: {
        bodyImage: {
            maxWidth: '75%',
        },
    }
}))

function Slide({
    data,
    width
}) {
    const classes = useStyles();
    const history = useHistory();

    // Background can either be an image, gradient, or solid color
    const backgroundStyle = useMemo(() => {
        if (!data.background) return null;
        if (data.background.image) {
            console.log('SETTIG BACKGROUND IMAGE', `url("${data.background.image}") no-repeat center center cover`)
            return {
                background: `url(${data.background.image}) no-repeat center center`,
                backgroundSize: 'cover',
            }
        }
        return data.background;
    }, [data])

    const toTitle = (title) => {
        if (!title) return null;
        let titleClasses = `${classes.titleCenter}`;
        if (title.style === 'pop') titleClasses += ` ${classes.textPop}`;
        return <Typography variant='h2' component='h1' className={titleClasses} style={{ color: title.color }}>{title.text}</Typography>
    }

    const toBodyChild = useCallback((child) => {
        if (!child) return null;
        // If child has 'content' property, then it is a GridItem
        if (Array.isArray(child.content)) {
            return <Grid className={classes.bodyGridItem} item xs={child.xs ?? 12} sm={child.sm}>
                {child.content.map(grandchild => toBodyChild(grandchild))}
            </Grid>
        }
        // If child has 'title' property, then it is a Typography
        if (child.title) return (
            <Typography
                variant={child.title.variant ?? 'h4'}
                component={child.title.component ?? 'h2'}
                textAlign={child.title.textAlign ?? 'left'}
                color={child.title.color}
            >
                {child.title.text}
            </Typography>
        )
        // If child has 'list' property, then it is a list of Typography
        if (child.list) return (
            <ul>
                {child.list.items.map(item => (
                    <li className={classes.bodyList} style={{color: item.color ?? child.list.color}}><Typography
                    variant={item.variant ?? child.list.variant ?? 'h4'}
                    component={item.component ?? child.list.component ?? 'h2'}
                    textAlign={item.textAlign ?? child.list.textAlign ?? 'left'}
                    color={item.color ?? child.list.color}
                >
                    {item.text}
                </Typography></li>
                ))}
            </ul>
        )
        // If child has 'video' property, then it is a YoutubeEmbed
        if (child.video) return <YoutubeEmbed embedId={child.video.link} width={Math.min(width - 16, 600)} height={Math.floor(Math.min(width - 16, 600) * 9 / 16)} />
        //TODO
        if (child.image) return <div className={classes.bodyImageContainer} ><img className={classes.bodyImage} alt={child.image.alt} src={child.image.src} /></div>
        //TODO
        return null;
    }, [width])

    const toBody = (body) => {
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
            onClick={() => button.link.includes('http') ? window.open(button.link, '_blank', 'noopener,noreferrer') : history.push(button.link)}
        >
            {button.text}
        </Button>
    }

    console.log('STYLES', backgroundStyle)

    return (
        <div className={classes.slideRoot} 
            style={{...backgroundStyle}}>
            <div className={classes.slidePad}>
                {toTitle(data?.title)}
                {toBody(data?.body)}
                {toButton(data?.button)}
            </div>
        </div>
    );
}

export { Slide };