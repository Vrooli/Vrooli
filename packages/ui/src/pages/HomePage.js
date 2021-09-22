import React, { useCallback, useState, useEffect } from 'react';
import { 
    Hero,
    MissionSlide
} from 'components';
import { useQuery } from '@apollo/client';
import { imagesByLabelQuery } from 'graphql/query';
import { IMAGE_SIZE, IMAGE_USE, SERVER_URL } from '@local/shared';
import { getImageSrc } from 'utils';

function HomePage() {
    // Load all images on home page
    const [imageData, setImageData] = useState([]);
    const { data: currImages } = useQuery(imagesByLabelQuery, { variables: { label: IMAGE_USE.Home } });

    useEffect(() => {
        // Table data must be extensible, and needs position
        setImageData(currImages?.imagesByLabel?.map((image, index) => ({
            ...image,
            src: `${SERVER_URL}/${getImageSrc(image, IMAGE_SIZE.XL)}`,
            pos: index
        })));
    }, [currImages])

    const getImage = useCallback((pos) => {
        return imageData?.length > pos ? imageData[pos] : null
    }, [imageData])

    // Slides in order
    const slides = [Hero, MissionSlide]

    return (
        <div>
            {slides.map((Slide, index) => <Slide key={`slide-${index}`} image={getImage(index)} />)}
        </div >
    );
}

HomePage.propTypes = {
    
}

export { HomePage };