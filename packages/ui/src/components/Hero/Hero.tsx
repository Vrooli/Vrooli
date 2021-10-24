// Code inspired by https://github.com/rmolinamir/hero-slider
import { useCallback, useEffect, useState } from 'react';
import { useHistory } from 'react-router-dom';
import { Typography, Button } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { LINKS } from 'utils';
import { Slider } from './Slider'
import { imagesByLabelQuery } from 'graphql/query';
import { useQuery } from '@apollo/client';
import { imagesByLabel, imagesByLabelVariables, imagesByLabel_imagesByLabel } from 'graphql/generated/imagesByLabel';

const useStyles = makeStyles(() => ({
    hero: {
        position: 'relative',
        overflow: 'hidden',
        pointerEvents: 'none',
        height: '100vh',
        width: '100vw',
    },
    contentWrapper: {
        position: 'absolute',
        top: '0',
        left: '0',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        flexFlow: 'column',
        width: '100%',
        height: '100%',
        margin: '0',
        padding: '0',
        pointerEvents: 'none',
        backgroundColor: 'rgba(0, 0, 0, 0.1)'
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
    title: {
        margin: '0 auto',
        width: '90%'
    },
    subtitle: {
        margin: '24px auto 0',
        width: '80%'
    },
    mainButton: {
        pointerEvents: 'auto'
    }
}));

interface Props {
    text: string;
    subtext: string;
}

export const Hero = ({
    text,
    subtext,
}: Props) => {
    let history = useHistory();
    const classes = useStyles();
    const [images, setImages] = useState<imagesByLabel_imagesByLabel[]>([]);
    const { data } = useQuery<imagesByLabel, imagesByLabelVariables>(imagesByLabelQuery, { variables: { label: 'hero' } });
    useEffect(() => {
        setImages(data?.imagesByLabel ?? []);
    }, [data])

    const toShopping = useCallback(() => history.push(LINKS.Shopping), [history]);

    return (
        <div className={classes.hero}>
            <Slider images={images} autoPlay={true} />
            <div className={classes.contentWrapper}>
                <Typography variant='h2' component='h1' className={classes.title + ' ' + classes.textPop}>{text}</Typography>
                <Typography variant='h4' component='h2' className={classes.subtitle + ' ' + classes.textPop}>{subtext}</Typography>
                <Button
                    type="submit"
                    color="secondary"
                    className={classes.mainButton}
                    onClick={toShopping}
                >
                    Request Quote
                </Button>
            </div>
        </div>
    );
};
