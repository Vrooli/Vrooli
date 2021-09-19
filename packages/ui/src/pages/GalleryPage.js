import React, { useEffect, useState } from 'react';
import Carousel from 'react-gallery-carousel';
import 'react-gallery-carousel/dist/index.css';
import { InformationalBreadcrumbs } from 'components';
import { getImageSrc, PUBS, PubSub } from 'utils';
import { makeStyles } from '@material-ui/styles';
import { imagesByLabelQuery } from 'graphql/query';
import { useQuery } from '@apollo/client';
import { IMAGE_SIZE, SERVER_URL } from '@local/shared';
import { useTheme } from '@emotion/react';

const useStyles = makeStyles(() => ({
    imageList: {
        spacing: 0,
    },
    tileImg: {

    },
    popupMain: {
        display: 'flex',
        width: '100%',
        height: '100%',
    },
    popupImg: {
        maxHeight: '90vh',
        maxWidth: '100%',
        display: 'block',
        borderRadius: '10px',
        objectFit: 'contain',
    },
    carousel: {
        width: '100%',
        height: 'calc(100vw * 0.8)'
    }
}));

function GalleryPage() {
    const classes = useStyles();
    const theme = useTheme();
    const [images, setImages] = useState([]);
    const { data: imageData, error } = useQuery(imagesByLabelQuery, { variables: { label: 'gallery' } });

    if (error) PubSub.publish(PUBS.Snack, { message: error.message ?? 'Unknown error occurred', severity: 'error', data: error });

    useEffect(() => {
        if (!Array.isArray(imageData?.imagesByLabel)) {
            setImages([]);
            return;
        }
        setImages(imageData.imagesByLabel.map((data) => ({ 
            alt: data.alt, 
            src: `${SERVER_URL}/${getImageSrc(data)}`, 
            thumbnail: `${SERVER_URL}/${getImageSrc(data, IMAGE_SIZE.M)}`
        })))
    }, [imageData])

    // useHotkeys('escape', () => setCurrImg([null, null]));
    // useHotkeys('arrowLeft', () => navigate(-1));
    // useHotkeys('arrowRight', () => navigate(1));

    return (
        <div id='page'>
            <InformationalBreadcrumbs textColor={theme.palette.secondary.dark} />
            <Carousel className={classes.carousel} canAutoPlay={false} images={images} />
        </div>
    );
}

GalleryPage.propTypes = {
}

export { GalleryPage };