import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { BUSINESS_NAME, LINKS } from "@local/consts";
import { AppBar, Box, Stack, useTheme } from "@mui/material";
import { forwardRef, useCallback, useEffect, useMemo } from "react";
import { noSelect } from "../../../styles";
import { useDimensions } from "../../../utils/hooks/useDimensions";
import { useIsLeftHanded } from "../../../utils/hooks/useIsLeftHanded";
import { useWindowSize } from "../../../utils/hooks/useWindowSize";
import { useLocation } from "../../../utils/route";
import { Header } from "../../text/Header/Header";
import { HideOnScroll } from "../HideOnScroll/HideOnScroll";
import { NavbarLogo } from "../NavbarLogo/NavbarLogo";
import { NavList } from "../NavList/NavList";
export const Navbar = forwardRef(({ shouldHideTitle = false, title, help, below, }, ref) => {
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { dimensions, ref: dimRef } = useDimensions();
    const toHome = useCallback(() => setLocation(LINKS.Home), [setLocation]);
    const scrollToTop = useCallback(() => window.scrollTo({ top: 0, behavior: "smooth" }), []);
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const { logoState } = useMemo(() => {
        const logoState = (isMobile && title) ? "icon" : "full";
        return { logoState };
    }, [isMobile, title]);
    const isLeftHanded = useIsLeftHanded();
    useEffect(() => {
        if (!title)
            return;
        document.title = `${title} | ${BUSINESS_NAME}`;
    }, [title]);
    const logo = useMemo(() => (_jsx(Box, { onClick: toHome, sx: {
            padding: 0,
            display: "flex",
            alignItems: "center",
            marginRight: isMobile && isLeftHanded ? 1 : "auto",
            marginLeft: isMobile && isLeftHanded ? "auto" : 1,
        }, children: _jsx(NavbarLogo, { onClick: toHome, state: logoState }) })), [isLeftHanded, isMobile, logoState, toHome]);
    return (_jsxs(Box, { id: 'navbar', ref: ref, sx: { paddingTop: `${Math.max(dimensions.height, 64)}px` }, children: [_jsx(HideOnScroll, { children: _jsxs(AppBar, { onClick: scrollToTop, ref: dimRef, sx: {
                        ...noSelect,
                        background: palette.primary.dark,
                        minHeight: "64px!important",
                        position: "fixed",
                        zIndex: 300,
                    }, children: [_jsxs(Stack, { direction: "row", spacing: 0, alignItems: "center", sx: {
                                paddingLeft: 1,
                                paddingRight: 1,
                            }, children: [!(isMobile && isLeftHanded) ? logo : _jsx(Box, { sx: {
                                        marginRight: "auto",
                                        maxHeight: "100%",
                                    }, children: _jsx(NavList, {}) }), isMobile && title && _jsx(Header, { help: help, title: title }), (isMobile && isLeftHanded) ? logo : _jsx(Box, { sx: {
                                        marginLeft: "auto",
                                        maxHeight: "100%",
                                    }, children: _jsx(NavList, {}) })] }), isMobile && below] }) }), !isMobile && title && !shouldHideTitle && _jsx(Header, { help: help, title: title }), !isMobile && below] }));
});
//# sourceMappingURL=Navbar.js.map