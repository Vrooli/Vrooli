import React, { useCallback, useState, useEffect } from 'react';
import { Slide } from 'components';
import { useQuery } from '@apollo/client';
import { imagesByLabelQuery } from 'graphql/query';
import { IMAGE_USE, SERVER_URL } from '@local/shared';
import { getImageSrc, LINKS } from 'utils';
import { makeStyles } from '@material-ui/styles';

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

function MissionPage() {
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
            id: 'the-problem',
            title: { text: 'The Problem', position: 'center' },
            body: [
                {
                    xs: 12,
                    content: [
                        { title: { text: 'talk about governance' } },
                        { title: { text: 'governance is influenced by money' } },
                        { title: { text: 'best way to make money is through entrepreneurship' } },
                        { title: { text: 'barriers to entrepreneurship are time, money, and knowledge' } },
                    ]
                }
            ],
        },
        {
            id: 'our-mission',
            title: { text: 'Our Mission', position: 'center' },
            body: [
                {
                    xs: 12,
                    content: [
                        { title: { text: 'Deliver the power of entrepreneurship to the masses' } },
                        { title: { text: 'something about how reducing the cost, labor, and knowledge required to profit from your ideas will be low enough for all to enjoy' } },
                    ]
                }
            ],
        },
        {
            id: 'some-assembly-required',
            title: { text: 'Some Assembly Required', position: 'center' },
            body: [
                {
                    xs: 12,
                    content: [
                        { title: { text: 'something about a system to automate entrepreneurship would be too complicated to create. Instead, creating a platform that it can emerge from' } },
                    ]
                }
            ],
        },
        {
            id: 'welcome-to-vrooli',
            title: { text: 'Welcome to Vrooli', position: 'center' },
            body: [
                {
                    xs: 12,
                    content: [
                        { title: { text: 'talk about how Vrooli aims to build a future-friendly, open-source platform for automating entrepreneurship, and how automation will reduce the cost, labor, and knowledge required.' } },
                    ]
                }
            ],
            button: { text: `Learn More`, position: 'center', link: 'https://matthalloran8.medium.com/the-next-generation-of-global-collaboration-a4839766e29e#e1eb' }
        },
        {
            id: 'whats-the-catch',
            title: { text: `What's the Catch?`, position: 'center' },
            body: [
                {
                    xs: 12,
                    content: [
                        { title: { text: 'We at Vrooli believe in free software - free to use, study, and change.' } },
                        { title: { text: 'Vrooli will (hopefully) be funded through Project Catalyst and donations. We aim to handle compute and maintenance costs by connecting routines to smart contracts on the Cardano blockchain.' } },
                        { title: { text: `We encourage users to promote and contribute the project and it's mission. If we are successful, we may just save the world` } },
                    ]
                }
            ],
        },
        {
            id: 'roadmap',
            title: { text: 'Roadmap - Q1 2022', position: 'center' },
            body: [
                
            ],
        },
        {
            id: 'roadmap-q2-2022',
            title: { text: 'Q2 2022', position: 'center' },
            body: [
                
            ],
        },
        {
            id: 'roadmap-q3-2022',
            title: { text: 'Q3 2022', position: 'center' },
            body: [
                
            ],
        },
        {
            id: 'roadmap-q4-2022',
            title: { text: 'Q4 2022', position: 'center' },
            body: [
                
            ],
        },
        {
            id: 'roadmap-2023-plus',
            title: { text: '2023 and Beyond', position: 'center' },
            body: [
                
            ],
        },
    ]

    return (
        <div className={classes.root}>
            {slides.map((data, index) => <Slide key={`slide-${index}`} width={width} data={data} />)}
        </div >
    );
}

MissionPage.propTypes = {
    
}

export { MissionPage };