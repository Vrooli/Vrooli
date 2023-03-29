import { Box, BoxProps, Grid, IconButton, Stack, Tooltip, Typography } from "@mui/material";
import { CSSProperties } from "@mui/styles";
import { LINKS, SOCIALS, WHITE_PAPER_URL } from "@shared/consts";
import { ArticleIcon, DiscordIcon, GitHubIcon, PlayIcon, TwitterIcon } from "@shared/icons";
import { openLink, useLocation } from "@shared/route";
import Earth from 'assets/img/Earth.svg';
import MonkeyCoin from 'assets/img/monkey-coin-page.png';
import { PulseButton } from "components/buttons/PulseButton/PulseButton";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SlideContainer, SlideContainerNeon, SlideContent, SlidePage } from "components/slides";
import { TwinkleStars } from "components/TwinkleStars/TwinkleStars";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { greenNeonText, iconButtonProps, slideImageContainer, slideText, slideTitle, textPop } from "styles";
import { LandingViewProps } from "../types";

interface GlossyContainerProps extends BoxProps {
    children: React.ReactNode;
    sx?: { [key: string]: any };
}

/**
 * A semi-transparent container with a glossy (blur) effect
 */
const GlossyContainer = ({
    children,
    sx,
    ...props
}: GlossyContainerProps) => {
    return (
        <Box
            sx={{
                boxShadow: '0px 0px 6px #040505',
                backgroundColor: 'rgba(255,255,255,0.3)',
                backdropFilter: 'blur(24px)',
                borderRadius: '0.5rem',
                padding: 2,
                height: '100%',
                ...sx
            }}
            {...props}
        >
            {children}
        </Box>
    );
};

// Used for scroll snapping and url hash
const slide1Id = "open-source-economy"
const slide2Id = "three-steps"
const slide3Id = "freedom"
const slide4Id = "share"
const slide5Id = "automate"
const slide6Id = "sky-is-limit"
const slide7Id = "get-started"
const slideContentIds = [slide1Id, slide2Id, slide3Id, slide4Id, slide5Id, slide6Id, slide7Id];

/**
 * View displayed for Home page when not logged in
 */
export const LandingView = ({
    display = 'page'
}: LandingViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Track if earth/sky is in view, and hndle scroll snap on slides
    const [earthTransform, setEarthTransform] = useState<string>('translate(0%, 100%) scale(1)');
    const [isSkyVisible, setIsSkyVisible] = useState<boolean>(false);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);
    const lasScrollPosRef = useRef<number>(window.pageYOffset || document.documentElement.scrollTop);
    const currScrollPosRef = useRef<number>(window.pageYOffset || document.documentElement.scrollTop);
    const scrollDirectionRef = useRef<'up' | 'down'>('down');
    useEffect(() => {
        const onScroll = () => {
            // console.log('scrolling', document.body.scrollTop, document.body.offsetTop)
            const scrollPos = window.pageYOffset || document.documentElement.scrollTop;
            if (scrollPos !== currScrollPosRef.current) {
                scrollDirectionRef.current = scrollPos > currScrollPosRef.current ? 'down' : 'up';
                lasScrollPosRef.current = currScrollPosRef.current;
                currScrollPosRef.current = scrollPos;
            }
            // Helper to check if an element is in view
            const inView = (element: HTMLElement | null) => {
                if (!element) return false;
                const rect = element.getBoundingClientRect();
                const windowHeight = (window.innerHeight || document.documentElement.clientHeight);
                return rect.top < windowHeight / 2;
            }
            // Use slides 6 and 7 to determine earth position and sky visibility
            const earthHorizonSlide = document.getElementById(slide6Id);
            const earthFullSlide = document.getElementById(slide7Id);
            if (inView(earthFullSlide)) {
                setEarthTransform('translate(25%, 25%) scale(0.8)');
                setIsSkyVisible(true);
            } else if (inView(earthHorizonSlide)) {
                setEarthTransform('translate(0%, 69%) scale(1)');
                setIsSkyVisible(true);
            } else {
                setEarthTransform('translate(0%, 100%) scale(1)');
                setIsSkyVisible(false);
            }
            // Create timeout to snap to nearest slide
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
            timeoutRef.current = setTimeout(() => {
                const slides = slideContentIds.map(id => document.getElementById(id));
                // console.log('slides', slides)
                let minDistance = Infinity;
                let nearestSlide;

                for (const slide of slides) {
                    if (!slide) continue;
                    // Calculate distance from slide to scroll position
                    const rect = slide.getBoundingClientRect();
                    const windowHeight = (window.innerHeight || document.documentElement.clientHeight);
                    const distance = Math.abs(rect.top - windowHeight / 2);
                    // If slide is at least halfway in view, set with min distance of 0
                    if (rect.top < windowHeight / 2 && rect.bottom > windowHeight / 2) {
                        minDistance = 0;
                        nearestSlide = slide;
                    }
                    // Otherwise, determine closest scroll using direction and distance
                    else if (
                        (scrollDirectionRef.current === 'down' && rect.top > 0) ||
                        (scrollDirectionRef.current === 'up' && rect.bottom < windowHeight)
                    ) {
                        if (distance < minDistance) {
                            minDistance = distance;
                            nearestSlide = slide;
                        }
                    }
                }
                nearestSlide?.scrollIntoView({ behavior: 'smooth' });
            }, 350);
        }
        // Add scroll listener to body
        window.addEventListener('scroll', onScroll, { passive: true });
        return () => {
            window.removeEventListener('scroll', onScroll);
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
        }
    }, []);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
            />
            <SlidePage id="landing-slides" sx={{ background: 'radial-gradient(circle, rgb(6 6 46) 12%, rgb(1 1 36) 52%, rgb(3 3 20) 80%)' }}>
                <SlideContainerNeon id="neon-container" show={!isSkyVisible} sx={{ zIndex: 3 }}>
                    <SlideContent id={slide1Id}>
                        <Typography component="h1" sx={{
                            ...slideTitle,
                            ...greenNeonText,
                            fontFamily: 'Neuropol',
                            fontWeight: 'bold',
                        }}>
                            An Open-Source Economy
                        </Typography>
                        <Grid container spacing={2}>
                            <Grid item xs={12} margin="auto" sx={{ marginTop: 4, marginBottom: 4 }}>
                                <Typography component="h2" variant="h5" sx={{ ...slideText, textAlign: 'center' }}>
                                    We're building the tools to automate the future of work
                                </Typography>
                            </Grid>
                        </Grid>
                        <PulseButton
                            variant="outlined"
                            color="secondary"
                            onClick={() => openLink(setLocation, LINKS.Start)}
                            startIcon={<PlayIcon fill='#0fa' />}
                            sx={{
                                marginLeft: 'auto !important',
                                marginRight: 'auto !important',
                            }}
                        >Start</PulseButton>
                        {/* Icon buttons for White paper, GitHub, Twitter, and Discord */}
                        <Stack direction="row" spacing={2} display="flex" justifyContent="center" alignItems="center" sx={{ paddingTop: 8, zIndex: 3 }}>
                            <Tooltip title="Read the white Paper" placement="bottom">
                                <IconButton onClick={() => openLink(setLocation, WHITE_PAPER_URL)} sx={iconButtonProps}>
                                    <ArticleIcon fill='#0fa' />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title="Check out our code" placement="bottom">
                                <IconButton onClick={() => openLink(setLocation, SOCIALS.GitHub)} sx={iconButtonProps}>
                                    <GitHubIcon fill='#0fa' />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title="Follow us on Twitter" placement="bottom">
                                <IconButton onClick={() => openLink(setLocation, SOCIALS.Twitter)} sx={iconButtonProps}>
                                    <TwitterIcon fill='#0fa' />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title="Join us on Discord" placement="bottom">
                                <IconButton onClick={() => openLink(setLocation, SOCIALS.Discord)} sx={iconButtonProps}>
                                    <DiscordIcon fill='#0fa' />
                                </IconButton>
                            </Tooltip>
                        </Stack>
                    </SlideContent>
                    <SlideContent id={slide2Id}>
                        <Typography variant='h2' mb={4} sx={{ ...slideTitle }}>Three Easy Steps</Typography>
                        <Grid container spacing={2} mb={4}>
                            <Grid item xs={12} sm={4}>
                                <GlossyContainer>
                                    <Typography variant='h5' mb={2}><b>Connect</b></Typography>
                                    <ul style={{ textAlign: 'left' }}>
                                        <li>Find a routine for anything you want to accomplish</li>
                                        <li>Fly solo or join an organization</li>
                                    </ul>
                                </GlossyContainer>
                            </Grid>
                            <Grid item xs={12} sm={4}>
                                <GlossyContainer>
                                    <Typography variant='h5' mb={2}><b>Collaborate</b></Typography>
                                    <ul style={{ textAlign: 'left' }}>
                                        <li>Build, vote, and give feedback on routines</li>
                                        <li>Design the ultimate organization</li>
                                    </ul>
                                </GlossyContainer>
                            </Grid>
                            <Grid item xs={12} sm={4}>
                                <GlossyContainer>
                                    <Typography variant='h5' mb={2}><b>Automate</b></Typography>
                                    <ul style={{ textAlign: 'left' }}>
                                        <li>Connect to APIs and smart contracts</li>
                                        <li>Complete complex tasks from a single site</li>
                                    </ul>
                                </GlossyContainer>
                            </Grid>
                        </Grid>
                        <Typography variant="h5" sx={{ ...slideText, textAlign: 'center' }}>
                            This combination creates a <Box sx={{ display: 'contents', color: 'gold' }}><b>self-improving productivity machine</b></Box>
                        </Typography>
                    </SlideContent>
                    <SlideContent id={slide3Id}>
                        <Typography variant='h2' mb={4} sx={{ ...slideTitle }}>
                            Take Back Your Freedom
                        </Typography>
                        <Grid container spacing={2}>
                            <Grid item xs={12} sm={6} margin="auto">
                                <Stack direction="column" spacing={2}>
                                    <Typography variant="h5" sx={{ ...slideText }}>
                                        Vrooli empowers you to unlock personal growth, streamline tasks, and leverage your skills effectively.
                                    </Typography>
                                    <Typography variant="h5" sx={{ ...slideText }}>
                                        It guides you through organizing your life, acquiring new knowledge, and monetizing your talents.
                                    </Typography>
                                </Stack>
                            </Grid>
                            <Grid item xs={12} sm={6} sx={{ paddingLeft: '0 !important' }}>
                                <Box sx={{ ...slideImageContainer }}>
                                    <img alt="Routine runner interface example - Minting tokens" src={MonkeyCoin} />
                                </Box>
                            </Grid>
                        </Grid>
                    </SlideContent>
                    <SlideContent id={slide4Id}>
                        <Typography variant="h2" sx={{ ...slideTitle }}>
                            Sharing is Scaling
                        </Typography>
                        <Grid container spacing={2}>
                            <Grid item xs={12} sm={6} sx={{ paddingLeft: '0 !important' }}>
                                <Box sx={{ ...slideImageContainer }}>
                                    <img alt="Routine runner interface example - Minting tokens" src={MonkeyCoin} />
                                </Box>
                            </Grid>
                            <Grid item xs={12} sm={6} margin="auto">
                                <Stack direction="column" spacing={2}>
                                    <Typography variant="h5" sx={{ ...slideText }}>
                                        Utilize existing routines as building blocks to rapidly construct complex processes.
                                    </Typography>
                                    <Typography variant="h5" sx={{ ...slideText }}>
                                        Contribute to the community to receive rewards, recognition, and valuable feedback.
                                    </Typography>
                                </Stack>
                            </Grid>
                        </Grid>
                    </SlideContent>
                    <SlideContent id={slide5Id}>
                        <Typography variant="h2" sx={{ ...slideTitle }}>
                            Automate With Minimal Effort
                        </Typography>
                        <Grid container spacing={2}>
                            <Grid item xs={12} margin="auto" sx={{ marginTop: 4, marginBottom: 4 }}>
                                <Typography component="h2" variant="h5" sx={{ ...slideText, textAlign: 'center' }}>
                                    Harness the power of AI to generate routines, streamlining processes and maximizing productivity.
                                </Typography>
                            </Grid>
                        </Grid>
                    </SlideContent>
                </SlideContainerNeon>
                <SlideContainer id='sky-slide' sx={{ color: 'white', background: 'black', zIndex: 4 }}>
                    {/* Background stars */}
                    <TwinkleStars
                        amount={400}
                        sx={{
                            position: 'absolute',
                            pointerEvents: 'none',
                            top: 0,
                            left: 0,
                            width: '100%',
                            height: '100%',
                            zIndex: 4,
                            background: 'black',
                            opacity: isSkyVisible ? 1 : 0,
                            transition: 'opacity 1s ease-in-out',
                        }}
                    />
                    {/* Earth at bottom of page. Changes position depending on the slide  */}
                    <Box
                        id="earth"
                        component="img"
                        src={Earth}
                        alt="Earth illustration"
                        sx={{
                            width: '150%',
                            position: 'fixed',
                            bottom: '0',
                            left: '-25%',
                            right: '-25%',
                            margin: 'auto',
                            maxWidth: '1000px',
                            maxHeight: '1000px',
                            transform: earthTransform,
                            transition: 'transform 1.5s ease-in-out',
                            zIndex: 5,
                        }}
                    />
                    <SlideContent id={slide6Id}>
                        <Typography variant='h2' mb={4} sx={{ ...slideTitle, zIndex: 6 }}>The Sky is the Limit</Typography>
                        <Typography variant="h5" sx={{ ...slideText, zIndex: 6 }}>
                            Our ultimate goal is to transition the world to a fully automated, post-capitalist economy. Here's how:
                        </Typography>
                        <ul style={{ textAlign: 'left', zIndex: 6 }}>
                            <li>Foster a decentralized, collaborative AI ecosystemt</li>
                            <li>Prioritize ethical and socially responsible AI development</li>
                            <li>Democratize access to AI-driven automation for all</li>
                        </ul>
                    </SlideContent>
                    <SlideContent id={slide7Id}>
                        <Typography variant="h2" mb={4} sx={{ ...slideTitle, ...textPop, zIndex: 6 } as CSSProperties}>
                            Ready to Change the World?
                        </Typography>
                        <PulseButton
                            variant="outlined"
                            color="secondary"
                            onClick={() => openLink(setLocation, LINKS.Start)}
                            startIcon={<PlayIcon fill='#0fa' />}
                            sx={{
                                marginLeft: 'auto !important',
                                marginRight: 'auto !important',
                                zIndex: 6,
                            }}
                        >{t('Start')}</PulseButton>
                    </SlideContent>
                </SlideContainer>
            </SlidePage >
        </>
    )
}