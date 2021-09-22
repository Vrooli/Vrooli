import React, { useCallback, useState, useEffect } from 'react';
import { Slide } from 'components';
import { useQuery } from '@apollo/client';
import { imagesByLabelQuery } from 'graphql/query';
import { IMAGE_USE, SERVER_URL } from '@local/shared';
import { getImageSrc, LINKS } from 'utils';

function HomePage() {
    // Load all images on home page
    const [imageData, setImageData] = useState([]);
    const [width, setWidth] = useState(window.innerWidth);
    const { data: currImages } = useQuery(imagesByLabelQuery, { variables: { label: IMAGE_USE.Home } });

    // Auto-updates width. Used to determine what size image to fetch
    useEffect(() => {
        const onResize = window.addEventListener('resize', () => setWidth(window.innerWidth))
        return () => {
            window.removeEventListener('resize', onResize)
        }
    })

    useEffect(() => {
        // Table data must be extensible, and needs position
        setImageData(currImages?.imagesByLabel?.map((image, index) => ({
            ...image,
            src: `${SERVER_URL}/${getImageSrc(image, width)}`,
            pos: index
        })));
    }, [currImages])

    const getImage = useCallback((pos) => {
        return imageData?.length > pos ? imageData[pos] : null
    }, [imageData])

    // Slides in order
    const slides = [
        {
            label: 'Hero',
            title: { text: 'Your portal to idea monetization', position: 'center', style: 'pop' },
            button: { text: 'Join Waitlist', position: 'center', color: 'secondary', link: LINKS.Waitlist }
        },
        {
            label: 'Mission',
            title: { text: 'Our Mission', position: 'center' },
            body: [
                {
                    xs: 6,
                    content: [{ title: { text: 'Bring the power of entrepreneurship to the masses, through the use of visual work routines' } }]
                },
                {
                    xs: 6,
                    content: [{ image: { src: '', width: '', height: '', objectFit: '' } }]
                }
            ],
            button: { text: 'Learn More', position: 'center', color: 'secondary', link: LINKS.Mission }
        }
    ]

    return (
        <div>
            {slides.map((data, index) => <Slide key={`slide-${index}`} data={{...data, background: { image: getImage(index)} }} />)}
        </div >
    );
}

HomePage.propTypes = {
    
}

export { HomePage };