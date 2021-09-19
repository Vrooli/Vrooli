import React from 'react';
import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles({
    slider: props => ({
        transform: `translateX(-${props.translate}px)`,
        transition: `transform ease-out ${props.transition}ms`,
        height: '100%',
        width: `${props.width}px`,
        display: 'flex',
    }),
});

const SliderContent = ({
    translate,
    transition,
    width,
    children
}) => {
    const classes = useStyles({translate, transition, width});
    return (
        <div className={classes.slider}>
            {children}
        </div>
    )
}

export { SliderContent }