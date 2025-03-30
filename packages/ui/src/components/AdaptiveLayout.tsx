import { noop } from "@local/shared";
import { Box, SwipeableDrawer, SwipeableDrawerProps, styled, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Routes } from "../Routes.js";
import { SessionContext } from "../contexts/session.js";
import { useIsLeftHanded } from "../hooks/subscriptions.js";
import { useMenu } from "../hooks/useMenu.js";
import { useWindowSize } from "../hooks/useWindowSize.js";
import { useLocation } from "../route/router.js";
import { LayoutComponentId, LayoutPositionId, useLayoutStore } from "../stores/LayoutStore.js";
import { ELEMENT_IDS, Z_INDEX } from "../utils/consts.js";
import { PubSub } from "../utils/pubsub.js";
import { SiteNavigator } from "./navigation/SiteNavigator.js";

// Drawer sizes and limits
const LEFT_DRAWER_WIDTH_DEFAULT_PX = 280;
const RIGHT_DRAWER_WIDTH_DEFAULT_PX = 280;
const MIN_LEFT_DRAWER_WIDTH = 150;
const MAX_LEFT_DRAWER_WIDTH = 500;
const MIN_RIGHT_DRAWER_WIDTH = 150;
const MAX_RIGHT_DRAWER_WIDTH = 500;

const leftDrawerPaperProps = { id: ELEMENT_IDS.LeftDrawer } as const;
const rightDrawerPaperProps = { id: ELEMENT_IDS.RightDrawer } as const;
const drawerModalProps = { sx: { zIndex: Z_INDEX.Drawer } } as const;

interface ResizableDrawerProps extends SwipeableDrawerProps {
    size: number,
}
const ResizableDrawer = styled(SwipeableDrawer)<ResizableDrawerProps>(({ size }) => ({
    width: size,
    flexShrink: 0,
    "& .MuiDrawer-paper": {
        width: size,
        boxSizing: "border-box",
    },
}));

interface ComponentProps {
    id: LayoutComponentId;
    /**
     * Optional story to render instead of the default routes.
     * Used for testing.
     */
    Story?: React.ComponentType;
    target: HTMLElement | null;
}

/**
 * Generic component wrapper that renders its content into a target element via portal
 */
function PortalComponent({ id, Story, target }: ComponentProps) {
    const session = useContext(SessionContext);

    // Get the appropriate component based on ID
    function getComponent() {
        switch (id) {
            case "navigator":
                return <SiteNavigator />;
            case "primary":
                return Story ? <Story /> : <Routes sessionChecked={session !== undefined} />;
            // case "secondary":
            //     return <ChatHistoryComponent />;
            default:
                return <Box>Unknown component</Box>;
        }
    }

    // Component content
    const content = (
        <Box height="100%" display="flex" flexDirection="column" overflow="hidden">
            {getComponent()}
        </Box>
    );

    // Render content into the target via portal if provided, otherwise return null
    return target ? createPortal(content, target) : null;
}

/**
 * Main Adaptive Layout Component
 * Manages the layout with drawers and integrates with router
 */
export function AdaptiveLayout({ Story }: { Story?: React.ComponentType }) {
    const { breakpoints } = useTheme();
    const [location] = useLocation();
    const isLeftHanded = useIsLeftHanded();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    // Get layout state from store
    const {
        positions,
        routeControlledPosition,
        swapLeftAndMain,
        swapMainAndRight,
        swapLeftAndRight,
        setRouteControlledComponent,
    } = useLayoutStore();

    // Handler drawer state and close
    const { isOpen: isLeftDrawerOpen, close: closeLeftDrawer } = useMenu({
        id: ELEMENT_IDS.LeftDrawer,
        isMobile,
    });
    const { isOpen: isRightDrawerOpen, close: closeRightDrawer } = useMenu({
        id: ELEMENT_IDS.RightDrawer,
        isMobile,
    });
    useEffect(function publishMenuOpenOnViewportChange() {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.LeftDrawer, isOpen: isLeftDrawerOpen });
        PubSub.get().publish("menu", { id: ELEMENT_IDS.RightDrawer, isOpen: isRightDrawerOpen });
    }, [breakpoints, isLeftDrawerOpen, isRightDrawerOpen]);
    const handleCloseLeftDrawer = useCallback(() => { closeLeftDrawer(); }, [closeLeftDrawer]);
    const handleCloseRightDrawer = useCallback(() => { closeRightDrawer(); }, [closeRightDrawer]);

    // Refs for portal target containers
    const leftContainerRef = useRef<HTMLDivElement | null>(null);
    const mainContainerRef = useRef<HTMLDivElement | null>(null);
    const rightContainerRef = useRef<HTMLDivElement | null>(null);

    // Force re-render when switching between mobile and desktop
    const [, setToggle] = useState(false);
    useEffect(() => {
        setToggle(t => !t);
    }, [isMobile]);

    useEffect(function determineRouteControlledComponentEffect() {
        // Example URL structure: /component-id
        // e.g., /chat, /profile, /tasks
        const path = location.pathname.substring(1); // Remove leading slash

        if (path) {
            const componentId = path as LayoutComponentId;
            // Only update if it's a valid component ID
            if (["chat", "profile", "tasks"].includes(componentId)) {
                setRouteControlledComponent(routeControlledPosition, componentId);
            }
        }
    }, [location.pathname, routeControlledPosition, setRouteControlledComponent]);

    // Get the target element for each component based on its position
    function getTargetForComponent(id: LayoutComponentId): HTMLElement | null {
        const position = Object.entries(positions).find(([, compId]) => compId === id)?.[0] as LayoutPositionId;

        if (position === "left") return leftContainerRef.current;
        if (position === "main") return mainContainerRef.current;
        if (position === "right") return rightContainerRef.current;
        return null;
    }

    // Determine drawer anchors based on handedness
    const leftDrawerAnchor = isLeftHanded ? "right" : "left";
    const rightDrawerAnchor = isLeftHanded ? "left" : "right";

    // State and refs for resizing
    const [leftDrawerWidth, setLeftDrawerWidth] = useState(LEFT_DRAWER_WIDTH_DEFAULT_PX);
    const [rightDrawerWidth, setRightDrawerWidth] = useState(RIGHT_DRAWER_WIDTH_DEFAULT_PX);
    const [isResizingLeft, setIsResizingLeft] = useState(false);
    const [isResizingRight, setIsResizingRight] = useState(false);
    const leftStartXRef = useRef(0);
    const leftStartWidthRef = useRef(0);
    const rightStartXRef = useRef(0);
    const rightStartWidthRef = useRef(0);

    // Handlers for left drawer resize
    function handleLeftMouseDown(e: React.MouseEvent<HTMLDivElement>) {
        setIsResizingLeft(true);
        leftStartXRef.current = e.clientX;
        leftStartWidthRef.current = leftDrawerWidth;
        e.preventDefault();
    }

    useEffect(function handleLeftDrawerResizeEffect() {
        function handleMouseMove(e: MouseEvent) {
            if (!isResizingLeft) return;
            const dx = e.clientX - leftStartXRef.current;
            const newWidth =
                leftDrawerAnchor === "left"
                    ? leftStartWidthRef.current + dx
                    : leftStartWidthRef.current - dx;
            const clampedWidth = Math.min(
                Math.max(newWidth, MIN_LEFT_DRAWER_WIDTH),
                MAX_LEFT_DRAWER_WIDTH,
            );
            setLeftDrawerWidth(clampedWidth);
        }
        function handleMouseUp() {
            if (isResizingLeft) {
                setIsResizingLeft(false);
            }
        }
        window.addEventListener("mousemove", handleMouseMove);
        window.addEventListener("mouseup", handleMouseUp);
        return () => {
            window.removeEventListener("mousemove", handleMouseMove);
            window.removeEventListener("mouseup", handleMouseUp);
        };
    }, [isResizingLeft, leftDrawerAnchor]);

    // Handlers for right drawer resize
    function handleRightMouseDown(e: React.MouseEvent<HTMLDivElement>) {
        setIsResizingRight(true);
        rightStartXRef.current = e.clientX;
        rightStartWidthRef.current = rightDrawerWidth;
        e.preventDefault();
    }

    useEffect(function handleRightDrawerResizeEffect() {
        function handleMouseMove(e: MouseEvent) {
            if (!isResizingRight) return;
            const dx = e.clientX - rightStartXRef.current;
            const newWidth =
                rightDrawerAnchor === "right"
                    ? rightStartWidthRef.current - dx
                    : rightStartWidthRef.current + dx;
            const clampedWidth = Math.min(
                Math.max(newWidth, MIN_RIGHT_DRAWER_WIDTH),
                MAX_RIGHT_DRAWER_WIDTH,
            );
            setRightDrawerWidth(clampedWidth);
        }
        function handleMouseUp() {
            if (isResizingRight) {
                setIsResizingRight(false);
            }
        }
        window.addEventListener("mousemove", handleMouseMove);
        window.addEventListener("mouseup", handleMouseUp);
        return () => {
            window.removeEventListener("mousemove", handleMouseMove);
            window.removeEventListener("mouseup", handleMouseUp);
        };
    }, [isResizingRight, rightDrawerAnchor]);

    return (
        <Box id={ELEMENT_IDS.AdaptiveLayout} display="flex" height="100vh" width="100%">
            {/* Hidden container for mounting all components */}
            <Box display="none">
                {/* Components are always mounted but rendered into different targets */}
                <PortalComponent id="navigator" target={getTargetForComponent("navigator")} />
                <PortalComponent id="primary" Story={Story} target={getTargetForComponent("primary")} />
                <PortalComponent id="secondary" target={getTargetForComponent("secondary")} />
            </Box>

            {/* What the user sees */}
            <ResizableDrawer
                size={isMobile ? leftDrawerWidth : (isLeftDrawerOpen ? leftDrawerWidth : 0)}
                anchor={leftDrawerAnchor}
                ModalProps={drawerModalProps}
                open={isLeftDrawerOpen}
                onOpen={noop}
                onClose={handleCloseLeftDrawer}
                PaperProps={leftDrawerPaperProps}
                variant={isMobile ? "temporary" : "persistent"}
            >
                <Box position="relative" height="100%">
                    <Box ref={leftContainerRef} overflow="auto" flexGrow={1} />
                    {/* Only enable resizing on non-mobile */}
                    {!isMobile && (
                        <Box
                            onMouseDown={handleLeftMouseDown}
                            sx={{
                                position: "absolute",
                                top: 0,
                                bottom: 0,
                                // Place handle on the appropriate edge based on anchor
                                [leftDrawerAnchor === "left" ? "right" : "left"]: 0,
                                width: "5px",
                                cursor: "ew-resize",
                                zIndex: 1,
                            }}
                        />
                    )}
                </Box>
            </ResizableDrawer>
            <Box ref={mainContainerRef} overflow="auto" flexGrow={1} />
            {/* <Box
                ref={mainContainerRef}
                overflow="auto"
                flexGrow={1}
                sx={{
                    transition: transitions.create("margin", {
                        easing: transitions.easing.sharp,
                        duration: drawerTransitionDuration.enter,
                    }),
                    // marginLeft: !isMobile ? (isLeftDrawerOpen ? leftDrawerWidth : 0) : 0,
                    // marginRight: !isMobile ? (isRightDrawerOpen ? rightDrawerWidth : 0) : 0,
                }}
            /> */}
            <ResizableDrawer
                size={isMobile ? rightDrawerWidth : (isRightDrawerOpen ? rightDrawerWidth : 0)}
                anchor={rightDrawerAnchor}
                ModalProps={drawerModalProps}
                open={isRightDrawerOpen}
                onOpen={noop}
                onClose={handleCloseRightDrawer}
                PaperProps={rightDrawerPaperProps}
                variant={isMobile ? "temporary" : "persistent"}
            >
                <Box position="relative" height="100%">
                    <Box ref={rightContainerRef} overflow="auto" flexGrow={1} />
                    {!isMobile && (
                        <Box
                            onMouseDown={handleRightMouseDown}
                            sx={{
                                position: "absolute",
                                top: 0,
                                bottom: 0,
                                // For right drawer, place handle on the opposite side of its anchor
                                [rightDrawerAnchor === "right" ? "left" : "right"]: 0,
                                width: "5px",
                                cursor: "ew-resize",
                                zIndex: 1,
                            }}
                        />
                    )}
                </Box>
            </ResizableDrawer>
        </Box >
    );
} 
