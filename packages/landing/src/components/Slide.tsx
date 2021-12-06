import { useCallback, useMemo } from 'react';
import {
    Button,
    Grid,
    Theme,
    Typography,
} from '@material-ui/core';
import { useHistory } from 'react-router';
import { YoutubeEmbed } from 'components';
import { makeStyles } from '@material-ui/styles';
import { openLink } from 'utils';

const useStyles = makeStyles((theme: Theme) => ({
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
        padding: theme.spacing(4),
        zIndex: 1,
    },
    titleCenter: {
        textAlign: 'center',
        paddingLeft: '5vw',
        paddingRight: '5vw',
        paddingBottom: '1em',
    },
    mainGrid: {
        alignItems: 'center',
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
    bodyVideo: {
        display: 'flex',
        justifyContent: 'center',
    },
    buttonsContainer: {
        display: 'flex',
        margin: 'auto',
        maxWidth: '75%',
    },
    button: {
        height: '100%',
    },
    textPop: {
        padding: '0',
        color: 'white',
        textAlign: 'center',
        fontWeight: 600,
        textShadow:
            `-1px -1px 0 black,  
                1px -1px 0 black,
                -1px 1px 0 black,
                1px 1px 0 black`
    },
    [theme.breakpoints.down(500)]: {
        slidePad: {
            padding: '2px'
        }
    },
    [theme.breakpoints.down('sm')]: {
        slidePad: {
            padding: theme.spacing(2),
        },
        bodyImage: {
            maxWidth: '75%',
        },
        buttonsContainer: {
            maxWidth: '100%'
        }
    }
}))

export const Slide = ({
    data,
    width
}) => {
    const classes = useStyles();
    const history = useHistory();

    // Background can either be an image, gradient, or solid color
    const backgroundStyle = useMemo(() => {
        if (!data.background) return null;
        if (data.background.image) {
            return {
                // position: 'fixed',
                // top: '0',
                // left: '0',
                width: '100%',
                height: '100%',
                background: `url(${data.background.image}) no-repeat center center ${data.background.fixed ? 'fixed' : ''}`,
                backgroundSize: 'cover',
                //@ts-ignore
                backgroundSize: '100% 100%',
                WebkitBackgroundSize: 'cover',
                MozBackgroundSize: 'cover',
                OBackgroundSize: 'cover',
                backgroundAttachment: 'scroll !important',
            }
        }
        return data.background;
    }, [data])

    const toTitle = (title) => {
        if (!title) return null;
        let titleClasses = `${classes.titleCenter}`;
        if (title.style === 'pop') titleClasses += ` ${classes.textPop}`;
        return <Typography variant={title.variant ?? 'h3'} component='h1' className={titleClasses} style={{ color: title.color, overflowWrap: title.overflowWrap ?? 'anywhere' }}>{title.text}</Typography>
    }

    const toButton = useCallback((button) => {
        if (!button) return null;
        let buttonClasses = `${classes.button}`;
        return <Button
            fullWidth
            className={buttonClasses}
            type="submit"
            size="large"
            color={button.color ?? 'secondary'}
            onClick={() => openLink(history, button.link)}
        >
            {button.text}
        </Button>
    }, [classes.button, history])


    const toButtonsGrid = useCallback((buttons, sm) => {
        return <div className={classes.buttonsContainer}>
            <Grid container spacing={1}>
                {buttons.map(b => (
                    <Grid item xs={12} sm={sm}>{toButton(b)}</Grid>
                ))}
            </Grid>
        </div>
    }, [classes.buttonsContainer, toButton])

    const toBodyChild = useCallback((child) => {
        if (!child) return null;
        // If child has 'content' property, then it is a GridItem
        if (Array.isArray(child.content)) {
            return <Grid className={classes.bodyGridItem} item xs={child.xs ?? 12} sm={child.sm} md={child.md}>
                {child.content.map(grandchild => toBodyChild(grandchild))}
            </Grid>
        }
        // If child has 'title' property, then it is a Typography
        if (child.title) return (
            <Typography
                variant={child.title.variant ?? 'h4'}
                component={child.title.component ?? 'h2'}
                textAlign={child.title.textAlign ?? 'left'}
                paddingBottom={child.title.paddingBottom ?? '0'}
                margin='auto'
                color={child.title.color}
                style={{ overflowWrap: child.title.overflowWrap ?? 'anywhere' }}
            >
                {child.title.text}
            </Typography>
        )
        // If child has 'list' property, then it is a list of Typography
        if (child.list) return (
            <ul style={{ paddingBottom: child.list.paddingBottom, margin: child.list.margin }}>
                {child.list.items.map(item => (
                    <li className={classes.bodyList} style={{ color: item.color ?? child.list.color }}><Typography
                        variant={item.variant ?? child.list.variant ?? 'h4'}
                        component={item.component ?? child.list.component ?? 'h2'}
                        textAlign={item.textAlign ?? child.list.textAlign ?? 'left'}
                        color={item.color ?? child.list.color}
                        style={{ overflowWrap: item.overflowWrap ?? child.list.overflowWrap ?? 'break-word' }}
                    >
                        {item.text}
                    </Typography></li>
                ))}
            </ul>
        )
        // If child has 'video' property, then it is a YoutubeEmbed
        if (child.video) return <YoutubeEmbed className={classes.bodyVideo} embedId={child.video.link} width={Math.min(width - 16, 600)} height={Math.floor(Math.min(width - 16, 600) * 9 / 16)} />
        if (child.image) return <div className={classes.bodyImageContainer} ><img className={classes.bodyImage} style={{ maxWidth: child.image.maxWidth }} alt={child.image.alt} src={child.image.src} /></div>
        if (child.buttons) {
            if (child.buttons.length === 1) return toButtonsGrid(child.buttons, 12);
            if (child.buttons.length === 2) return toButtonsGrid(child.buttons, 6);
            if (child.buttons.length === 3) return toButtonsGrid(child.buttons, 4);
            if (child.buttons.length === 4) return toButtonsGrid(child.buttons, 3);
            return null;
        }
        return null;
    }, [classes.bodyGridItem, classes.bodyImage, classes.bodyImageContainer, classes.bodyList, classes.bodyVideo, toButtonsGrid, width])

    const toBody = (body) => {
        if (!Array.isArray(body)) return null;
        return <Grid className={classes.mainGrid} container spacing={1}>
            {body.map(child => toBodyChild(child))}
        </Grid>
    }

    return (
        <div id={data?.id} key={data?.id}
            className={classes.slideRoot}
            style={{ ...backgroundStyle }}>
            <div className={classes.slidePad}>
                {toTitle(data?.title)}
                {toBody(data?.body)}
            </div>
        </div>
    );
}