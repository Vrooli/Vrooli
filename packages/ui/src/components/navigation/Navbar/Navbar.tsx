import { useCallback, useMemo } from 'react';
import { APP_LINKS } from '@shared/consts';
import { AppBar, Box, useTheme, Stack } from '@mui/material';
import { NavList } from '../NavList/NavList';
import { useLocation } from '@shared/route';
import { NavbarProps } from '../types';
import { HideOnScroll } from '..';
import { noSelect } from 'styles'
import { PageTitle } from 'components/text';
import { NavbarLogo } from '../NavbarLogo/NavbarLogo';
import { NavbarLogoState } from '../types';
import { useDimensions, useWindowSize } from 'utils';

/**
 * Navbar displayed at the top of the page. Has a few different 
 * looks depending on data passed to it.
 * 
 * If the screen is large, the navbar is always displayed the same. In 
 * this case, the title and other content are displayed below the navbar.
 * 
 * Otherwise, the default look is logo & business name on the left, and 
 * account menu profile icon on the right.
 * 
 * If title data is passed in, the business name is hidden. The 
 * title is displayed in the middle, with a help icon if specified.
 * 
 * Content to display below the title (but still in the navbar) can also 
 * be passed in. This is useful for displaying a search bar, page tabs, etc. This 
 * content is inside the navbar on small screens, and below the navbar on large screens.
 */
export const Navbar = ({
    session,
    shouldHideTitle = false,
    title,
    help,
    below,
}: NavbarProps) => {
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();

    const { dimensions, ref } = useDimensions();

    const toHome = useCallback(() => setLocation(APP_LINKS.Home), [setLocation]);
    const scrollToTop = useCallback(() => window.scrollTo({ top: 0, behavior: 'smooth' }), []);

    // Determine display texts and states
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const { logoState } = useMemo(() => {
        const logoState: NavbarLogoState = (isMobile && title) ? 'icon' : 'full';
        return { logoState };
    }, [isMobile, title]);

    const logo = useMemo(() => (
        <Box
            onClick={toHome}
            sx={{
                padding: 0,
                display: 'flex',
                alignItems: 'center',
                marginLeft: 1,
                marginRight: 'auto',
            }}
        >
            <NavbarLogo
                onClick={toHome}
                state={logoState}
            />
        </Box>
    ), [logoState, toHome]);

    return (
        <Box sx={{ paddingTop: `${Math.max(dimensions.height, 64)}px` }}>
            <HideOnScroll>
                <AppBar
                    onClick={scrollToTop}
                    ref={ref}
                    sx={{
                        ...noSelect,
                        background: palette.primary.dark,
                        minHeight: '64px!important',
                        position: 'fixed', // Allows items to be displayed below the navbar
                        zIndex: 100,
                    }}>
                    {/* <Toolbar> */}
                    <Stack direction="row" spacing={0} alignItems="center" sx={{
                        paddingLeft: 1,
                        paddingRight: 1,
                    }}>
                        {logo}
                        {/* Title displayed here on mobile */}
                        {isMobile && title && <PageTitle help={help} title={title} />}
                        <Box sx={{
                            marginLeft: 'auto',
                            maxHeight: '100%',
                        }}>
                            <NavList session={session} />
                        </Box>
                    </Stack>
                    {/* "below" displayed inside AppBar on mobile */}
                    {isMobile && below}
                    {/* </Toolbar> */}
                </AppBar>
            </HideOnScroll>
            {/* Title displayed here on desktop */}
            {!isMobile && title && !shouldHideTitle && <PageTitle
                help={help}
                title={title}
            />}
            {/* "below" and title displayered here on desktop */}
            {!isMobile && below}
        </Box>
    );
}