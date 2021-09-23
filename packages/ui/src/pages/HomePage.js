import React, { useCallback, useState, useEffect } from 'react';
import { Slide } from 'components';
import { useQuery } from '@apollo/client';
import { imagesByLabelQuery } from 'graphql/query';
import { IMAGE_USE, SERVER_URL } from '@local/shared';
import { getImageSrc, LINKS } from 'utils';
import { makeStyles } from '@material-ui/styles';
import Relax from 'assets/img/relax.png';
import BlankRoutine from 'assets/img/blank-routine-1.png';
import MonkeyCoin from 'assets/img/monkey-coin-page.png';
import Community from 'assets/img/community.svg';

// const useStyles = makeStyles((theme) => ({
//     root: {
//         paddingTop: '0',
//     },
//     [theme.breakpoints.down(500)]: {
//         root: {
//             paddingTop: 'calc(14vh - 10px)',
//         }
//     },
// }))

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
            title: { text: 'Entrepreneurship - Without the Hustle!', position: 'center', color: 'white' },
            // background: { background: 'linear-gradient(344deg, rgba(34,81,149,1) 0%, rgba(17,119,168,1) 46%, rgba(74,152,180,1) 75%)' },
            background: { image: getImage(0)?.src },
            body: [
                {
                    xs: 12, sm: 6,
                    content: [{ title: { text: 'Getting things done has never been easier, through the power of visual work routines', textAlign: 'center', color: 'white' } }]
                },
                {
                    xs: 12, sm: 6,
                    content: [{ image: { src: Relax, alt: 'Man relaxing at beach - by Vecteezy' } }]
                }
            ],
            button: { text: 'Learn More', position: 'center', color: 'secondary', link: LINKS.Mission }
        },
        {
            title: { text: 'A What Routine?', position: 'center' },
            background: { background: '#d7deef' },
            body: [
                {
                    xs: 12,
                    content: [
                        { title: { text: 'A visual work routine is an intuitive process for completing a specific task - similar to a flowchart', textAlign: 'center' } },
                    ]
                },
                {
                    xs: 12, sm: 6,
                    content: [
                        { title: { text: 'Visual work routines transform entrepreneurship into a process that is:', variant: 'h5' } },
                        { list: { items: [{ text: 'Approachable' }, { text: 'Transparent'}, { text: 'Automatable' }], variant: 'h5' }}
                    ]
                },
                {
                    xs: 12, sm: 6,
                    content: [{ image: { src: BlankRoutine, alt: 'Non-descriptive visual work routine' } }]
                }
            ],
        },
        // {
        //     title: { text: 'Our Mission', position: 'center' },
        //     body: [
        //         {
        //             xs: 12,
        //             content: [
        //                 { title: { text: 'Transform entrepreneurship into a process that is:' } },
        //                 { list: { items: [{ text: 'Approachable' }, { text: 'Transparent'}, { text: 'Automatable' }] }}
        //             ]
        //         }
        //     ],
        //     button: { text: 'Learn More', position: 'center', color: 'secondary', link: LINKS.Mission }
        // },
        {
            title: { text: 'Understand Your Workflow', position: 'center', color: 'white' },
            background: { background: 'linear-gradient(344deg, rgba(5,4,34,1) 0%, rgba(16,31,142,1) 75%, rgba(59,102,154,1) 75%)' },
            body: [
                {
                    xs: 12,
                    content: [{ title: { text: `Routines may consist of subroutines, and each subroutine may also have its own routines`, textAlign: 'center', color: 'white' } }]
                },
                {
                    xs: 12,
                    content: [{ title: { text: `This hierarchy simplifies the process of building routines, and allows you to inspect each part of your workflow in as much abstraction as you'd like!`, textAlign: 'center', color: 'white' } }]
                },
            ],
            button: { text: 'Join Waitlist', position: 'center', color: 'secondary', link: LINKS.Waitlist }
        },
        {
            title: { text: 'Say Goodbye to Endless Browser Tabs👋' },
            background: { background: '#d7deef' },
            body: [
                {
                    xs: 12,
                    content: [
                        { title: { text: 'Auto-generated interfaces unlock the possibility of performing entire routines through Vrooli', textAlign: 'center' } },
                    ]
                },
                {
                    xs: 12, sm: 6,
                    content: [{ image: { src: MonkeyCoin, alt: 'Non-descriptive visual work routine' } }]
                },
                {
                    xs: 12, sm: 6,
                    content: [
                        { title: { text: 'Benefits of this approach include:', variant: 'h5' } },
                        { list: { items: [{ text: 'Reduced context switching' }, { text: 'Increased focus and organization'}], variant: 'h5' }}
                    ]
                },
            ],
            button: { text: 'How does it work?', position: 'center', link: 'https://matthalloran8.medium.com/the-next-generation-of-global-collaboration-a4839766e29e#82db' }
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
            title: { text: 'AI & Blockchain Support' },
            background: { background: '#d7deef' },
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
            background: { image: getImage(1)?.src, fixed: true },
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