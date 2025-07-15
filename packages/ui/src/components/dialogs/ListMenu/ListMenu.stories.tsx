import type { Meta, StoryObj } from "@storybook/react";
import Button from "@mui/material/Button";
import { useRef, useState } from "react";
import { signedInNoPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ListMenu } from "./ListMenu.js";
import { type ListMenuItemData } from "../types.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof ListMenu> = {
    title: "Components/Dialogs/ListMenu",
    component: ListMenu,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 600,
            },
        },
        session: signedInNoPremiumWithCreditsSession,
    },
    tags: ["autodocs"],
    argTypes: {
        anchorEl: {
            control: false,
            description: "Element to anchor the menu to",
        },
        data: {
            control: { type: "object" },
            description: "Array of menu items",
        },
        id: {
            control: { type: "text" },
            description: "Unique ID for the menu",
        },
        title: {
            control: { type: "text" },
            description: "Optional title for the menu",
        },
        onSelect: {
            action: "item-selected",
            description: "Callback when item is selected",
        },
        onClose: {
            action: "menu-closed",
            description: "Callback when menu is closed",
        },
    },
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Mock data for different menu types
const basicMenuData: ListMenuItemData<string>[] = [
    {
        value: "create",
        label: "Create New",
        iconInfo: { name: "Add" },
    },
    {
        value: "edit",
        label: "Edit",
        iconInfo: { name: "Edit" },
    },
    {
        value: "delete",
        label: "Delete",
        iconInfo: { name: "Delete" },
        iconColor: "#f44336",
    },
];

const advancedMenuData: ListMenuItemData<string>[] = [
    {
        value: "create",
        labelKey: "Create",
        iconInfo: { name: "Add" },
        helpData: {
            text: "Create a new item",
        },
    },
    {
        value: "edit",
        labelKey: "Edit",
        iconInfo: { name: "Edit" },
        helpData: {
            text: "Edit the selected item",
        },
    },
    {
        value: "copy",
        labelKey: "Copy",
        iconInfo: { name: "Copy" },
        helpData: {
            text: "Copy the selected item",
        },
    },
    {
        value: "share",
        labelKey: "Share",
        iconInfo: { name: "Share" },
        helpData: {
            text: "Share the selected item",
        },
    },
    {
        value: "delete",
        labelKey: "Delete",
        iconInfo: { name: "Delete" },
        iconColor: "#f44336",
        helpData: {
            text: "Permanently delete the selected item",
        },
    },
];

const longMenuData: ListMenuItemData<string>[] = [
    { value: "option1", label: "First Option", iconInfo: { name: "Star" } },
    { value: "option2", label: "Second Option", iconInfo: { name: "Bookmark" } },
    { value: "option3", label: "Third Option", iconInfo: { name: "Favorite" } },
    { value: "option4", label: "Fourth Option", iconInfo: { name: "Share" } },
    { value: "option5", label: "Fifth Option", iconInfo: { name: "Copy" } },
    { value: "option6", label: "Sixth Option", iconInfo: { name: "Edit" } },
    { value: "option7", label: "Seventh Option", iconInfo: { name: "Delete" } },
    { value: "option8", label: "Eighth Option", iconInfo: { name: "Add" } },
    { value: "option9", label: "Ninth Option", iconInfo: { name: "View" } },
    { value: "option10", label: "Tenth Option", iconInfo: { name: "Download" } },
];

// Showcase with controls
export const Showcase: Story = {
    render: (args) => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => {
            setAnchorEl(buttonRef.current);
        };

        const handleClose = () => {
            setAnchorEl(null);
            args.onClose?.();
        };

        const handleSelect = (value: string) => {
            console.log("Selected:", value);
            args.onSelect?.(value);
            handleClose();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="contained"
                >
                    Open Menu
                </Button>
                <ListMenu
                    {...args}
                    anchorEl={anchorEl}
                    onClose={handleClose}
                    onSelect={handleSelect}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Click the button above to open the menu</p>
                    <p>Menu State: {anchorEl ? "Open" : "Closed"}</p>
                </div>
            </div>
        );
    },
    args: {
        id: "showcase-menu",
        title: "Sample Menu",
        data: basicMenuData,
    },
};

// Basic menu without title
export const BasicMenu: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => setAnchorEl(buttonRef.current);
        const handleClose = () => setAnchorEl(null);
        const handleSelect = (value: string) => {
            console.log("Selected:", value);
            handleClose();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="outlined"
                >
                    Basic Menu
                </Button>
                <ListMenu
                    id="basic-menu"
                    anchorEl={anchorEl}
                    data={basicMenuData}
                    onClose={handleClose}
                    onSelect={handleSelect}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Simple menu without title</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Basic menu with simple items and no title.",
            },
        },
    },
};

// Menu with title
export const MenuWithTitle: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => setAnchorEl(buttonRef.current);
        const handleClose = () => setAnchorEl(null);
        const handleSelect = (value: string) => {
            console.log("Selected:", value);
            handleClose();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="outlined"
                >
                    Menu with Title
                </Button>
                <ListMenu
                    id="titled-menu"
                    title="Actions"
                    anchorEl={anchorEl}
                    data={basicMenuData}
                    onClose={handleClose}
                    onSelect={handleSelect}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Menu with title and close button</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Menu with a title header and close button.",
            },
        },
    },
};

// Advanced menu with help buttons
export const AdvancedMenu: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => setAnchorEl(buttonRef.current);
        const handleClose = () => setAnchorEl(null);
        const handleSelect = (value: string) => {
            console.log("Selected:", value);
            handleClose();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="outlined"
                >
                    Advanced Menu
                </Button>
                <ListMenu
                    id="advanced-menu"
                    title="Actions"
                    anchorEl={anchorEl}
                    data={advancedMenuData}
                    onClose={handleClose}
                    onSelect={handleSelect}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Menu with help buttons and translation keys</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Advanced menu with help buttons for each item and translation keys.",
            },
        },
    },
};

// Long menu (scrollable)
export const LongMenu: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => setAnchorEl(buttonRef.current);
        const handleClose = () => setAnchorEl(null);
        const handleSelect = (value: string) => {
            console.log("Selected:", value);
            handleClose();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="outlined"
                >
                    Long Menu
                </Button>
                <ListMenu
                    id="long-menu"
                    title="Many Options"
                    anchorEl={anchorEl}
                    data={longMenuData}
                    onClose={handleClose}
                    onSelect={handleSelect}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Menu with many items to test scrolling</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Menu with many items to test scrolling behavior.",
            },
        },
    },
};

// Empty menu
export const EmptyMenu: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => setAnchorEl(buttonRef.current);
        const handleClose = () => setAnchorEl(null);
        const handleSelect = (value: string) => {
            console.log("Selected:", value);
            handleClose();
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="outlined"
                >
                    Empty Menu
                </Button>
                <ListMenu
                    id="empty-menu"
                    title="No Items"
                    anchorEl={anchorEl}
                    data={[]}
                    onClose={handleClose}
                    onSelect={handleSelect}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Menu with no items</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Menu with no items to test empty state.",
            },
        },
    },
};

// Always open for design testing
export const AlwaysOpen: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);

        const handleClose = () => console.log("Menu closed");
        const handleSelect = (value: string) => console.log("Selected:", value);

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    variant="contained"
                    disabled
                >
                    Menu Anchor
                </Button>
                <ListMenu
                    id="always-open-menu"
                    title="Always Open Menu"
                    anchorEl={buttonRef.current}
                    data={basicMenuData}
                    onClose={handleClose}
                    onSelect={handleSelect}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Menu that stays open for design testing</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Menu that stays open for design and interaction testing.",
            },
        },
    },
};
