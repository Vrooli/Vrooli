import React from 'react';
import {
    Box,
    Button,
    Grid,
    Typography,
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { slideStyles } from './styles';
import { useHistory } from 'react-router';

const useStyles = makeStyles(slideStyles);

function Slide({
    data,
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

    const toBodyChild = (child) => {
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
        if (child.title) return <Typography variant={child.title.variant ?? 'h3'} component={child.title.component ?? 'h2'}>{child.title.text}</Typography>
        //TODO
        if (child.image) return null;
        //TODO
        return null;
    }

    const toBody = (body) => {
        console.log('in body', body)
        if (!Array.isArray(body)) return null;
        return <Grid container spacing={1}>
            {body.map(child => toBodyChild(child))}
        </Grid>
    }

    const toButton = (button) => {
        if (!button) return null;
        let buttonClasses = `${classes.buttonCenter}`;
        return <Button
            className={buttonClasses}
            type="submit"
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