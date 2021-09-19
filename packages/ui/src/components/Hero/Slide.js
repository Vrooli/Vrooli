import React from 'react';
import { memo } from 'react'
import { makeStyles } from '@material-ui/styles';
import { SERVER_URL } from '@local/shared';
import { getImageSrc } from 'utils';

const useStyles = makeStyles({
    slide: props => ({
        height: '100%',
        width: `${props.width}px`,
        objectFit: 'cover',
        overflow: 'hidden',
    }),
});

const Slide = memo(({ image, width }) => {
    const classes = useStyles({ width });
    return (
        <img className={classes.slide} src={image ? `${SERVER_URL}/${getImageSrc(image, width)}` : ''} alt={image?.alt ?? ''} />
    )
})

export { Slide };