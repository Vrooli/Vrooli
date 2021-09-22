import React, { useCallback, useState, useEffect } from 'react';
import { Slide } from 'components';
import { useQuery } from '@apollo/client';
import { imagesByLabelQuery } from 'graphql/query';
import { IMAGE_USE, SERVER_URL } from '@local/shared';
import { getImageSrc, LINKS } from 'utils';
import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles((theme) => ({
    root: {
        paddingTop: '0',
    },
    [theme.breakpoints.down(500)]: {
        root: {
            paddingTop: 'calc(14vh - 10px)',
        }
    },
}))

function HomePage() {
    const classes = useStyles();
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
            title: { text: 'Your portal to idea monetization', position: 'center', style: 'pop' },
            button: { text: 'Join Waitlist', position: 'center', color: 'secondary', link: LINKS.Waitlist }
        },
        {
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
        },
        {
            title: { text: 'Understand your workflow', position: 'center' },
            body: [
                {
                    xs: 12,
                    content: [{ title: { text: 'talk about how visualizing routines makes the whole process approachable to newbies', textAlign: 'center' } }]
                },
            ],
        },
        {
            title: { text: 'Intuitive interface', position: 'center' },
            body: [
                {
                    xs: 6,
                    content: [{ title: { text: 'talk about how routines can be created with drag n drop' } }]
                },
                {
                    xs: 6,
                    content: [{ title: { text: 'talk about how routines can be executed with simple UI' } }]
                },
            ],
        },
        {
            title: { text: 'Build With the Community' },
            body: [
                {
                    xs: 12,
                    content: [{ title: { text: 'talk about how you can use public routines instead of designing everything yourself', textAlign: 'center' } }]
                },
            ],
        },
        {
            title: { text: 'Say Goodbye to Endless Browser Tabs👋' },
            body: [
                {
                    xs: 12,
                    content: [{ title: { text: 'talk about how supported routines can be executed from Vrooli directly, to reduce context switching', textAlign: 'center' } }]
                },
            ],
        },
        {
            title: { text: 'AI & Blockchain Support' },
            body: [
                {
                    xs: 6,
                    content: [{ title: { text: 'talk about how it is public API, and how AIs are encouraged to execute routines' } }]
                },
                {
                    xs: 6,
                    content: [{ title: { text: 'talk about how routines can be connected to smart contracts on Cardano for automation' } }]
                },
            ],
        },
        {
            title: { text: 'Automatic Automation' },
            body: [
                {
                    xs: 12,
                    content: [{ title: { text: 'talk about how AI, smart contracts, and more specialized routines will allow everyones workflows to accelerate', textAlign: 'center' } }]
                },
            ],
        },
        {
            title: { text: 'Join the Movement' },
            body: [
                {
                    xs: 12,
                    content: [{ video: { link: 'Avyeo1f38Aw' } }]
                },
            ],
            button: { text: 'Join Waitlist', position: 'center', color: 'secondary', link: LINKS.Waitlist }
        },
    ]

    return (
        <div className={classes.root}>
            {slides.map((data, index) => <Slide key={`slide-${index}`} width={width} data={{...data, background: { image: getImage(index)} }} />)}
        </div >
    );
}

HomePage.propTypes = {
    
}

export { HomePage };