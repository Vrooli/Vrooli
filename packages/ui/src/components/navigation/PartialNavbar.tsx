import Box from "@mui/material/Box";
import { IconButton } from "../buttons/IconButton.js";
import { styled } from "@mui/material";
import { useCallback } from "react";
import { IconCommon } from "../../icons/Icons.js";
import { useLayoutStore } from "../../stores/layoutStore.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { PubSub } from "../../utils/pubsub.js";
import { NavbarInner } from "./Navbar.js";

const PartialHeaderBox = styled(Box)(() => ({
    display: "flex",
    justifyContent: "flex-end",
    padding: "8px",
}));

export interface PartialNavbarProps {
    /** Optional children to render in the navbar */
    children?: React.ReactNode;
    /** Optional color for the navbar text and icons */
    color?: string;
    /** Whether to keep the navbar visible when scrolling */
    keepVisible?: boolean;
}

/**
 * A simplified navbar for views displayed in the side menu.
 * Includes swap and close buttons to move the view to the main content area or close the side menu.
 */
export function PartialNavbar({
    children,
    color,
    keepVisible,
}: PartialNavbarProps) {
    const { swapMainAndRight } = useLayoutStore();

    const handleCloseDrawer = useCallback(() => {
        PubSub.get().publish("menu", { id: ELEMENT_IDS.RightDrawer, isOpen: false });
    }, []);

    const handleSwapView = useCallback(() => {
        swapMainAndRight();
    }, [swapMainAndRight]);

    return (
        <NavbarInner
            color={color}
            keepVisible={keepVisible}
        >
            <Box sx={{ display: "flex", flexGrow: 1, alignItems: "center" }}>
                {children}
            </Box>
            <PartialHeaderBox>
                <IconButton onClick={handleSwapView} variant="transparent" sx={{ mr: 1 }}>
                    <IconCommon name="MoveLeftRight" />
                </IconButton>
                <IconButton onClick={handleCloseDrawer} variant="transparent">
                    <IconCommon name="Close" />
                </IconButton>
            </PartialHeaderBox>
        </NavbarInner>
    );
} 
