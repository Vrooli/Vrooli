import React, { useCallback, useState, useEffect } from 'react';
import { Slide } from 'components';
import { useQuery } from '@apollo/client';
import { imagesByLabelQuery } from 'graphql/query';
import { IMAGE_USE, SERVER_URL } from '@local/shared';
import { getImageSrc, LINKS } from 'utils';
import { makeStyles } from '@material-ui/styles';
import BlankRoutine from 'assets/img/blank-routine-1.png';
import Community from 'assets/img/community.svg';

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
            background: { image: getImage(0)?.src },
            button: { text: 'Join Waitlist', position: 'center', color: 'secondary', link: LINKS.Waitlist }
        },
        {
            title: { text: 'Our Mission', position: 'center' },
            body: [
                {
                    xs: 12, sm: 6,
                    content: [{ title: { text: 'Bring the power of entrepreneurship to the masses, through the use of visual work routines' } }]
                },
                {
                    xs: 12, sm: 6,
                    content: [{ image: { src: BlankRoutine, alt: 'Non-descriptive visual work routine' } }]
                }
            ],
            button: { text: 'Learn More', position: 'center', color: 'secondary', link: LINKS.Mission }
        },
        {
            title: { text: 'Understand your workflow', position: 'center', color: 'white' },
            background: { background: 'linear-gradient(344deg, rgba(5,4,34,1) 0%, rgba(16,31,142,1) 75%, rgba(59,102,154,1) 75%)' },
            body: [
                {
                    xs: 12,
                    content: [{ title: { text: 'talk about how visualizing routines makes the whole process approachable to newbies', textAlign: 'center', color: 'white' } }]
                },
            ],
        },
        {
            title: { text: 'Intuitive interface', position: 'center' },
            body: [
                {
                    xs: 12, sm: 6,
                    content: [{ title: { text: 'talk about how routines can be created with drag n drop' } }]
                },
                {
                    xs: 12, sm: 6,
                    content: [{ title: { text: 'talk about how routines can be executed with simple UI' } }]
                },
            ],
        },
        {
            title: { text: 'Build With the Community', color: 'white' },
            background: { background: '#08125f' },
            body: [
                {
                    xs: 12, sm: 6,
                    content: [{ list: { color: 'white', items: [{ text: 'Expand on public routines' }, { text: 'Discuss and vote on routines'}, { text: 'Fund routine creation with Project Catalyst' }], textAlign: 'center' }}]
                },
                {
                    xs: 12, sm: 6,
                    content: [{ image: { src: Community, alt: 'Community illustration - by Vecteezy' } }]
                },
            ],
            button: { text: `What's Project Catalyst?`, position: 'center', link: 'https://matthalloran8.medium.com/the-next-generation-of-global-collaboration-a4839766e29e#4f79' }
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
                    xs: 12, sm: 6,
                    content: [{ title: { text: 'talk about how it is public API, and how AIs are encouraged to execute routines' } }]
                },
                {
                    xs: 12, sm: 6,
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
            title: { text: 'Join the Movement', style: 'pop' },
            background: { image: getImage(0)?.src },
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
            {slides.map((data, index) => <Slide key={`slide-${index}`} width={width} data={data} />)}
        </div >
    );
}

HomePage.propTypes = {
    
}

export { HomePage };