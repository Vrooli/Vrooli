import React from 'react';
import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles((theme) => ({
    dotContainer: {
        position: 'absolute',
        bottom: '25px',
        width: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        overflow: 'hidden',
    },
    dot: {
        padding: '10px',
        marginRight: '5px',
        cursor: 'pointer',
        borderRadius: '50%',
        opacity: '80%',
        border: '1px solid black',
    },
    active: {
        background: theme.palette.primary.main,
        opacity: '0.9',
    },
    inactive: {
        background: 'white',
    }
}));

const Dots = ({
    quantity = 0,
    activeIndex
}) => {
    const classes = useStyles();

    let slides = [];
    for (let i = 0; i < quantity; i++) {
        slides.push(<div key={'dot-'+i} className={`${classes.dot} ${activeIndex === i ? classes.active : classes.inactive}`} />)
    }

    return (
        <div className={classes.dotContainer}>
            {slides}
        </div>
    )
}

export { Dots };