import React, { useCallback, useState, useEffect } from 'react';
import { Slide } from 'components';
import { useQuery } from '@apollo/client';
import { imagesByLabelQuery } from 'graphql/query';
import { IMAGE_USE, SERVER_URL } from '@local/shared';
import { getImageSrc, LINKS } from 'utils';
import { makeStyles } from '@material-ui/styles';
import MattSketch from 'assets/img/thought-sketch-edited-3.png';

const useStyles = makeStyles((theme) => ({
    root: {
        paddingTop: 'calc(7vh - 10px)',
    },
    [theme.breakpoints.down(600)]: {
        root: {
            paddingTop: 'calc(14vh - 10px)',
        }
    },
}))

function AboutPage() {
    const classes = useStyles();
    // Load all images on page
    const [imageData, setImageData] = useState([]);
    const [width, setWidth] = useState(window.innerWidth);
    const { data: currImages } = useQuery(imagesByLabelQuery, { variables: { label: IMAGE_USE.Mission } });

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
            id: 'about-the-project',
            title: { text: 'About the Project', position: 'center' },
            body: [
                {
                    xs: 12,
                    content: [
                        { title: { text: 'something something project catalyst. link to article' } },
                    ]
                }
            ],
            button: { text: 'Learn More', position: 'center', color: 'secondary', link: LINKS.Mission }
        },
        {
            id: 'about-the-team',
            title: { text: 'About the Team', position: 'center' },
            body: [
                {
                    xs: 12, sm: 6,
                    content: [
                        { title: { text: 'about me', variant: 'h5' } },
                        { list: { items: [{ text: 'some' }, { text: 'things'}], variant: 'h5' }}
                    ]
                },
                {
                    xs: 12, sm: 6,
                    content: [{ image: { src: MattSketch, alt: 'Illustrated drawing of the founder of Vrooli - Matt Halloran' } }]
                },
            ],
            button: { text: 'Contact/Support', position: 'center', color: 'secondary', link: 'https://matthalloran.info/' }
        },
    ]

    return (
        <div className={classes.root}>
            {slides.map((data, index) => <Slide key={`slide-${index}`} width={width} data={data} />)}
        </div >
    );
}

AboutPage.propTypes = {
    
}

export { AboutPage };