import React, { useCallback, useState, useEffect } from 'react';
import { Slide } from 'components';
import { useQuery } from '@apollo/client';
import { imagesByLabelQuery } from 'graphql/query';
import { IMAGE_USE, SERVER_URL } from '@local/shared';
import { getImageSrc, LINKS } from 'utils';
import { makeStyles } from '@material-ui/styles';
import { Box } from '@material-ui/core';
import Relax from 'assets/img/relax.png';
import BlankRoutine from 'assets/img/blank-routine-1.png';
import MonkeyCoin from 'assets/img/monkey-coin-page.png';
import Community from 'assets/img/community.svg';
import Blockchain from 'assets/img/blockchain.png';

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
    // Load all images on page
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

    // Slides in orderbackground: rgb(8,24,79);
    const slides = [
        {
            id: 'get-things-done-easy',
            title: { text: 'Entrepreneurship - Without the Hustle!', position: 'center', color: 'white' },
            background: { background: 'linear-gradient(142deg, rgba(8,24,79,1) 16%, rgba(12,48,113,1) 61%, rgba(42,123,184,1) 93%)' },
            body: [
                {
                    sm: 6,
                    content: [{ title: { text: 'Getting things done has never been easier, through the power of visual work routines', textAlign: 'center', color: 'white' } }]
                },
                {
                    sm: 6,
                    content: [{ image: { src: Relax, alt: 'Man relaxing at beach - by Vecteezy' } }]
                }
            ],
            button: { text: 'Learn More', position: 'center', color: 'secondary', link: LINKS.Mission }
        },
        {
            id: 'routine-explanation',
            title: { text: (<React.Fragment>A <Box fontStyle="italic" display="inline">What</Box> Routine?</React.Fragment>), position: 'center' },
            background: { background: '#d7deef' },
            body: [
                {
                    content: [
                        { title: { text: 'A visual work routine is an intuitive process for completing a specific task - similar to a flowchart', textAlign: 'center' } },
                    ]
                },
                {
                    sm: 6,
                    content: [
                        { title: { text: 'Visual work routines transform entrepreneurship into a process that is:', variant: 'h5' } },
                        { list: { items: [{ text: 'Approachable' }, { text: 'Transparent'}, { text: 'Automatable' }], variant: 'h5' }}
                    ]
                },
                {
                    sm: 6,
                    content: [{ image: { src: BlankRoutine, alt: 'Non-descriptive visual work routine' } }]
                }
            ],
        },
        {
            id: 'understand-your-workflow',
            title: { text: 'Understand Your Workflow', position: 'center', color: 'white' },
            background: { background: 'linear-gradient(344deg, rgba(5,4,34,1) 0%, rgba(16,31,142,1) 75%, rgba(59,102,154,1) 75%)' },
            body: [
                {
                    content: [{ title: { text: `Routines may consist of subroutines, and each subroutine may also have its own routines`, textAlign: 'center', color: 'white' } }]
                },
                {
                    content: [{ title: { text: `This hierarchy simplifies the process of building routines, and allows you to inspect each part of your workflow in as much abstraction as you'd like!`, textAlign: 'center', color: 'white' } }]
                },
            ],
            button: { text: 'Join Waitlist', position: 'center', color: 'secondary', link: LINKS.Waitlist }
        },
        {
            id: 'auto-generated-interfaces',
            title: { text: 'Say Goodbye to Endless Browser Tabs👋' },
            background: { background: '#d7deef' },
            body: [
                {
                    content: [{ title: { text: 'Auto-generated interfaces unlock the possibility of performing entire routines without leaving Vrooli', textAlign: 'center' } }]
                },
                {
                    sm: 6,
                    content: [{ image: { src: MonkeyCoin, alt: 'Routine runner interface example - Minting tokens' } }]
                },
                {
                    sm: 6,
                    content: [
                        { title: { text: 'Benefits of this approach include:', variant: 'h4' } },
                        { list: { items: [{ text: 'Reduced context switching' }, { text: 'Increased focus and organization'}], variant: 'h5' }}
                    ]
                },
            ],
            button: { text: 'How does it work?', position: 'center', link: 'https://matthalloran8.medium.com/the-next-generation-of-global-collaboration-a4839766e29e#82db' }
        },
        {
            id: 'fund-your-idea',
            title: { text: 'Build With the Community', color: 'white' },
            background: { image: getImage(0)?.src },
            body: [
                {
                    sm: 6,
                    content: [{ list: { color: 'white', items: [{ text: 'Expand on public routines' }, { text: 'Discuss and vote on routines'}, { text: 'Fund routine creation with Project Catalyst' }], textAlign: 'center' }}]
                },
                {
                    sm: 6,
                    content: [{ image: { src: Community, alt: 'Community illustration - by Vecteezy' } }]
                },
            ],
            button: { text: `What's Project Catalyst?`, position: 'center', link: 'https://matthalloran8.medium.com/the-next-generation-of-global-collaboration-a4839766e29e#4f79' }
        },
        {
            id: 'blockchain-and-ai',
            title: { text: 'Blockchain and AI Support' },
            background: { background: '#d7deef' },
            body: [
                {
                    sm: 6,
                    content: [{ image: { src: Blockchain, alt: 'Blockchain illustration - by Vecteezy' } }]
                },
                {
                    sm: 6,
                    content: [
                        { title: { text: 'Routines can trigger smart contracts, bringing the power of the Cardano blockchain to your fingertips', paddingBottom: '32px' } },
                        { title: { text: 'Smart contracts allow for automated handling of resources - such as money and compute - drastically increasing the number of routines that can be run without human intervention' } }
                    ]
                },
            ],
        },
        {
            id: 'join-the-movement',
            title: { text: 'Join the Movement', style: 'pop' },
            background: { image: getImage(1)?.src, fixed: true },
            body: [
                {
                    content: [{ title: { text: `Be one of the first users when the project launches in Q1 2022. Let's change the world together!💙`, color: 'white', textAlign: 'center' } }]
                },
                {
                    content: [{ video: { link: 'Avyeo1f38Aw' } }]
                },
            ],
            buttons: [
                { text: 'Join Waitlist', position: 'center', color: 'secondary', link: LINKS.Waitlist },
                { text: 'View Proposal', position: 'center', color: 'secondary', link: 'https://cardano.ideascale.com/a/dtd/Community-Made-Interactive-Guides/367058-48088' },
                { text: 'Roadmap', position: 'center', color: 'secondary', link: LINKS.Roadmap }
            ]
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