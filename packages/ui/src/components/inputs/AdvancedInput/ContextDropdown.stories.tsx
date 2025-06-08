import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import { useCallback, useRef, useState } from "react";
import { ContextDropdown, type ListObject } from "./ContextDropdown.js";

const centerStyle = {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    minHeight: "100vh",
} as const;

const resultStyle = {
    position: "fixed",
    bottom: 20,
    left: "50%",
    transform: "translateX(-50%)",
    textAlign: "center",
} as const;

export default {
    title: "Components/Inputs/AdvancedInput/ContextDropdown",
    component: ContextDropdown,
};

// Wrapper component to handle the anchor element state
function DropdownWrapper({
    items = [],
    onSelect,
}: {
    items?: ListObject[];
    onSelect?: (item: ListObject) => void;
}) {
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const buttonRef = useRef<HTMLButtonElement>(null);

    const handleClick = useCallback(() => {
        setAnchorEl(buttonRef.current);
    }, []);

    const handleClose = useCallback(() => {
        setAnchorEl(null);
    }, []);

    const handleSelect = useCallback((item: ListObject) => {
        onSelect?.(item);
        if (item.type === "web") {
            handleClose();
        }
    }, [onSelect]);

    return (
        <Box sx={centerStyle}>
            <Button ref={buttonRef} onClick={handleClick} variant="contained">
                Open Dropdown
            </Button>
            <ContextDropdown
                anchorEl={anchorEl}
                onClose={handleClose}
                onSelect={handleSelect}
            />
        </Box>
    );
}

// Basic story with empty categories
export function Default() {
    return <DropdownWrapper />;
}
Default.parameters = {
    docs: {
        description: {
            story: "Shows the dropdown in its default state with empty categories.",
        },
    },
};

// Story with sample items in different categories
export function WithItems() {
    const [selectedItem, setSelectedItem] = useState<ListObject | null>(null);

    return (
        <>
            <DropdownWrapper onSelect={setSelectedItem} />
            {selectedItem && (
                <Box sx={resultStyle}>
                    Selected: {selectedItem.name} ({selectedItem.type})
                </Box>
            )}
        </>
    );
}
WithItems.parameters = {
    docs: {
        description: {
            story: "Demonstrates the dropdown with items and selection handling.",
        },
    },
};

// Story demonstrating web search selection
export function WebSearch() {
    const [searchInitiated, setSearchInitiated] = useState(false);

    return (
        <>
            <DropdownWrapper
                onSelect={(item) => {
                    if (item.type === "web") {
                        setSearchInitiated(true);
                    }
                }}
            />
            {searchInitiated && (
                <Box sx={resultStyle}>
                    Web search initiated
                </Box>
            )}
        </>
    );
}
WebSearch.parameters = {
    docs: {
        description: {
            story: "Shows how the web search option behaves differently from other selections.",
        },
    },
}; 
