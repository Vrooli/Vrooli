import { Box, Button, Drawer, useTheme } from "@mui/material";
import { ChevronLeftIcon, ChevronRightIcon } from "icons/common.js"; // Adjust import as needed
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { create } from "zustand";
import { useWindowSize } from "../../hooks/useWindowSize.js"; // Adjust import as needed

// --- Layout State Management with Zustand ---
interface LayoutState {
    left: string;
    main: string;
    right: string;
    swapLeftAndMain: () => void;
    swapMainAndRight: () => void;
    swapLeftAndRight: () => void;
}

const useLayoutStore = create<LayoutState>((set) => ({
    left: "ComponentA",
    main: "ComponentB",
    right: "ComponentC",
    swapLeftAndMain: () =>
        set((state) => ({
            left: state.main,
            main: state.left,
            right: state.right,
        })),
    swapMainAndRight: () =>
        set((state) => ({
            main: state.right,
            right: state.main,
            left: state.left,
        })),
    swapLeftAndRight: () =>
        set((state) => ({
            left: state.right,
            right: state.left,
            main: state.main,
        })),
}));

const modalProps = { keepMounted: true };

// --- Updated Test Component with Portal Support ---
function TestComponent({ name, target }) {
    const [counter, setCounter] = useState(0);

    useEffect(() => {
        console.log(`${name} mounted`);
        return () => console.log(`${name} unmounted`);
    }, [name]);

    function handleIncrement() {
        setCounter((c) => c + 1);
    }

    const content = (
        <Box p={2}>
            <h3>{name}</h3>
            <p>Counter: {counter}</p>
            <Button variant="contained" onClick={handleIncrement}>
                Increment
            </Button>
        </Box>
    );

    // Render content into the target via portal if provided, otherwise return null
    return target ? createPortal(content, target) : null;
}

// --- Main Layout Component ---
export function LayoutTest() {
    const { left, main, right, swapLeftAndMain, swapMainAndRight, swapLeftAndRight } =
        useLayoutStore();
    const theme = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= theme.breakpoints.values.md);
    const drawerWidth = 240;

    const [leftOpen, setLeftOpen] = useState(true);
    const [rightOpen, setRightOpen] = useState(true);

    // Refs for portal target containers
    const leftContainerRef = useRef(null);
    const mainContainerRef = useRef(null);
    const rightContainerRef = useRef(null);

    // Without this, the portal content will not be rendered until you click something, 
    // and the side drawers can lose content when switching between mobile and desktop.
    // Not sure how this works, but it's simple and effective.
    const [, setToggle] = useState(false);
    useEffect(() => {
        setToggle(t => !t);
    }, [isMobile]);

    // Map components to their current positions
    const positions = {
        ComponentA: left === "ComponentA" ? "left" : main === "ComponentA" ? "main" : "right",
        ComponentB: left === "ComponentB" ? "left" : main === "ComponentB" ? "main" : "right",
        ComponentC: left === "ComponentC" ? "left" : main === "ComponentC" ? "main" : "right",
    };

    function handleCloseLeftDrawer() {
        setLeftOpen(false);
    }

    function handleCloseRightDrawer() {
        setRightOpen(false);
    }

    return (
        <Box display="flex" height="100vh">
            {/* Hidden container for mounting all components */}
            <Box display="none">
                <TestComponent
                    key="ComponentA"
                    name="Component A"
                    target={
                        positions["ComponentA"] === "left"
                            ? leftContainerRef.current
                            : positions["ComponentA"] === "main"
                                ? mainContainerRef.current
                                : rightContainerRef.current
                    }
                />
                <TestComponent
                    key="ComponentB"
                    name="Component B"
                    target={
                        positions["ComponentB"] === "left"
                            ? leftContainerRef.current
                            : positions["ComponentB"] === "main"
                                ? mainContainerRef.current
                                : rightContainerRef.current
                    }
                />
                <TestComponent
                    key="ComponentC"
                    name="Component C"
                    target={
                        positions["ComponentC"] === "left"
                            ? leftContainerRef.current
                            : positions["ComponentC"] === "main"
                                ? mainContainerRef.current
                                : rightContainerRef.current
                    }
                />
            </Box>

            {/* --- Left Drawer --- */}
            <Drawer
                variant={isMobile ? "temporary" : "persistent"}
                open={leftOpen}
                onClose={handleCloseLeftDrawer}
                ModalProps={modalProps}
                PaperProps={{
                    style: { width: drawerWidth, boxSizing: "border-box" },
                }}
            >
                <Box height="100%" overflow="auto">
                    <Box
                        display="flex"
                        alignItems="center"
                        justifyContent="space-between"
                        p={1}
                        borderBottom="1px solid #ddd"
                    >
                        <Button onClick={handleCloseLeftDrawer}>
                            <ChevronLeftIcon />
                        </Button>
                        <span>Left Pane</span>
                    </Box>
                    {/* Container for portal content */}
                    <Box ref={leftContainerRef} />
                </Box>
            </Drawer>

            {/* --- Main Content Area --- */}
            <Box
                component="main"
                sx={{
                    flexGrow: 1,
                    p: 3,
                    transition: theme.transitions.create(["margin"], {
                        easing: theme.transitions.easing.sharp,
                        duration: theme.transitions.duration.leavingScreen,
                    }),
                    marginLeft: !isMobile && leftOpen ? `${drawerWidth}px` : 0,
                    marginRight: !isMobile && rightOpen ? `${drawerWidth}px` : 0,
                }}
            >
                <h1>Main Content</h1>
                {/* Container for portal content */}
                <Box ref={mainContainerRef} />
                <Box mt={2} display="flex" gap={1}>
                    <Button
                        variant="outlined"
                        onClick={() => setLeftOpen(true)}
                        disabled={leftOpen}
                    >
                        Open Left Drawer
                    </Button>
                    <Button
                        variant="outlined"
                        onClick={() => setRightOpen(true)}
                        disabled={rightOpen}
                    >
                        Open Right Drawer
                    </Button>
                </Box>
                <Box mt={2} display="flex" gap={1}>
                    <Button variant="outlined" onClick={swapLeftAndMain}>
                        Swap Left and Main
                    </Button>
                    <Button variant="outlined" onClick={swapMainAndRight}>
                        Swap Main and Right
                    </Button>
                    <Button variant="outlined" onClick={swapLeftAndRight}>
                        Swap Left and Right
                    </Button>
                </Box>
            </Box>

            {/* --- Right Drawer --- */}
            <Drawer
                anchor="right"
                variant={isMobile ? "temporary" : "persistent"}
                open={rightOpen}
                onClose={handleCloseRightDrawer}
                ModalProps={modalProps}
                PaperProps={{
                    style: { width: drawerWidth, boxSizing: "border-box" },
                }}
            >
                <Box height="100%" overflow="auto">
                    <Box
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "space-between",
                            p: 1,
                            borderBottom: "1px solid #ddd",
                        }}
                    >
                        <Button onClick={handleCloseRightDrawer}>
                            <ChevronRightIcon />
                        </Button>
                        <span>Right Pane</span>
                    </Box>
                    {/* Container for portal content */}
                    <Box ref={rightContainerRef} />
                </Box>
            </Drawer>
        </Box>
    );
}
