import { Typography } from "@mui/material";
import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { Button } from "../../buttons/Button.js";
import { Dialog, DialogActions, DialogContent } from "./Dialog.js";

const meta: Meta<typeof Dialog> = {
    title: "Components/Dialogs/Dialog",
    component: Dialog,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 600,
            },
        },
    },
    tags: ["autodocs"],
    decorators: [
        (Story) => (
            <div style={{
                minHeight: "100vh",
                backgroundColor: "transparent",
                padding: "20px",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
            }}>
                <Story />
            </div>
        ),
    ],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Sizes demonstration
export const Sizes: Story = {
    render: () => {
        const [openDialog, setOpenDialog] = useState<string | null>(null);
        const sizes = ["sm", "md", "lg", "xl", "full"] as const;

        return (
            <div style={{ display: "flex", gap: "16px", flexWrap: "wrap" }}>
                {sizes.map((size) => (
                    <Button
                        key={size}
                        onClick={() => setOpenDialog(size)}
                        variant="outline"
                    >
                        Open {size.toUpperCase()} Dialog
                    </Button>
                ))}

                {sizes.map((size) => (
                    <Dialog
                        key={size}
                        isOpen={openDialog === size}
                        onClose={() => setOpenDialog(null)}
                        title={`${size.toUpperCase()} Size Dialog`}
                        size={size}
                    >
                        <DialogContent>
                            <Typography>
                                This is a {size} sized dialog. Notice how the width changes based on the size prop.
                            </Typography>
                            {size === "full" && (
                                <Typography sx={{ mt: 2 }}>
                                    Full size dialogs take up the entire screen.
                                </Typography>
                            )}
                        </DialogContent>
                        <DialogActions>
                            <Button variant="primary" onClick={() => setOpenDialog(null)}>
                                Close
                            </Button>
                        </DialogActions>
                    </Dialog>
                ))}
            </div>
        );
    },
};

// Variants demonstration (button styling)
export const Variants: Story = {
    render: () => {
        const [openDialog, setOpenDialog] = useState<string | null>(null);
        const variants = ["default", "danger", "success", "space", "neon"] as const;

        return (
            <div style={{ display: "flex", gap: "16px", flexWrap: "wrap" }}>
                {variants.map((variant) => (
                    <Button
                        key={variant}
                        onClick={() => setOpenDialog(variant)}
                        variant={variant === "default" ? "primary" : variant === "space" || variant === "neon" ? variant : "outline"}
                    >
                        Open {variant} Dialog
                    </Button>
                ))}

                {variants.map((variant) => (
                    <Dialog
                        key={variant}
                        isOpen={openDialog === variant}
                        onClose={() => setOpenDialog(null)}
                        title={`${variant.charAt(0).toUpperCase() + variant.slice(1)} Action Dialog`}
                        variant={variant}
                    >
                        <DialogContent>
                            <Typography>
                                This dialog uses the theme colors for the dialog itself. The variant "{variant}"
                                affects how the action buttons are styled. Notice how the confirm button changes
                                based on the variant.
                            </Typography>
                        </DialogContent>
                        <DialogActions variant={variant}>
                            <Button
                                variant="ghost"
                                onClick={() => setOpenDialog(null)}
                            >
                                Cancel
                            </Button>
                            <Button
                                variant={variant === "danger" ? "danger" :
                                    variant === "success" ? "primary" :
                                        variant === "space" ? "space" :
                                            variant === "neon" ? "neon" : "primary"}
                                onClick={() => setOpenDialog(null)}
                            >
                                {variant === "danger" ? "Delete" : "Confirm"}
                            </Button>
                        </DialogActions>
                    </Dialog>
                ))}
            </div>
        );
    },
};

// Positions demonstration
export const Positions: Story = {
    render: () => {
        const [openDialog, setOpenDialog] = useState<string | null>(null);
        const positions = ["center", "top", "bottom", "left", "right"] as const;

        return (
            <div style={{ display: "flex", gap: "16px", flexWrap: "wrap" }}>
                {positions.map((position) => (
                    <Button
                        key={position}
                        onClick={() => setOpenDialog(position)}
                        variant="secondary"
                    >
                        Open {position} Dialog
                    </Button>
                ))}

                {positions.map((position) => (
                    <Dialog
                        key={position}
                        isOpen={openDialog === position}
                        onClose={() => setOpenDialog(null)}
                        title={`${position.charAt(0).toUpperCase() + position.slice(1)} Position`}
                        position={position}
                    >
                        <DialogContent>
                            <Typography>
                                This dialog is positioned at the {position} of the screen.
                            </Typography>
                        </DialogContent>
                        <DialogActions>
                            <Button variant="primary" onClick={() => setOpenDialog(null)}>
                                Close
                            </Button>
                        </DialogActions>
                    </Dialog>
                ))}
            </div>
        );
    },
};

// Without close button
export const NoCloseButton: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button onClick={() => setIsOpen(true)}>
                    Open Dialog without Close Button
                </Button>
                <Dialog
                    isOpen={isOpen}
                    onClose={() => setIsOpen(false)}
                    title="No Close Button"
                    showCloseButton={false}
                >
                    <DialogContent>
                        <Typography>
                            This dialog doesn't have a close button in the header.
                            You can still close it by clicking outside or using the action buttons.
                        </Typography>
                    </DialogContent>
                    <DialogActions>
                        <Button variant="primary" onClick={() => setIsOpen(false)}>
                            Got it
                        </Button>
                    </DialogActions>
                </Dialog>
            </>
        );
    },
};

// Prevent close on overlay click
export const NoOverlayClose: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button onClick={() => setIsOpen(true)}>
                    Open Dialog (No Overlay Close)
                </Button>
                <Dialog
                    isOpen={isOpen}
                    onClose={() => setIsOpen(false)}
                    title="Important Action"
                    closeOnOverlayClick={false}
                    variant="danger"
                >
                    <DialogContent>
                        <Typography>
                            This dialog cannot be closed by clicking outside.
                            You must use the buttons to close it. This demonstrates a
                            danger variant where the action buttons reflect the danger state.
                        </Typography>
                    </DialogContent>
                    <DialogActions variant="danger">
                        <Button variant="ghost" onClick={() => setIsOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="danger" onClick={() => setIsOpen(false)}>
                            Delete
                        </Button>
                    </DialogActions>
                </Dialog>
            </>
        );
    },
};

// Long content with scrolling
export const ScrollableContent: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button onClick={() => setIsOpen(true)}>
                    Open Dialog with Long Content
                </Button>
                <Dialog
                    isOpen={isOpen}
                    onClose={() => setIsOpen(false)}
                    title="Terms and Conditions"
                    size="lg"
                >
                    <DialogContent>
                        {Array.from({ length: 20 }, (_, i) => (
                            <Typography key={i} paragraph>
                                Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod
                                tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim
                                veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea
                                commodo consequat. Section {i + 1}.
                            </Typography>
                        ))}
                    </DialogContent>
                    <DialogActions>
                        <Button variant="ghost" onClick={() => setIsOpen(false)}>
                            Decline
                        </Button>
                        <Button variant="primary" onClick={() => setIsOpen(false)}>
                            Accept
                        </Button>
                    </DialogActions>
                </Dialog>
            </>
        );
    },
};

// Complex form example
export const FormDialog: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button onClick={() => setIsOpen(true)}>
                    Open Form Dialog
                </Button>
                <Dialog
                    isOpen={isOpen}
                    onClose={() => setIsOpen(false)}
                    title="Create New Project"
                    size="md"
                >
                    <DialogContent>
                        <div style={{ display: "flex", flexDirection: "column", gap: "24px" }}>
                            <div>
                                <label htmlFor="project-name" style={{ display: "block", marginBottom: 4 }}>
                                    Project Name
                                </label>
                                <input
                                    id="project-name"
                                    type="text"
                                    placeholder="Enter project name"
                                    style={{
                                        width: "100%",
                                        padding: "8px 12px",
                                        borderRadius: 4,
                                        border: "1px solid #e0e0e0",
                                    }}
                                />
                            </div>
                            <div>
                                <label htmlFor="project-description" style={{ display: "block", marginBottom: 4 }}>
                                    Description
                                </label>
                                <textarea
                                    id="project-description"
                                    placeholder="Enter project description"
                                    rows={4}
                                    style={{
                                        width: "100%",
                                        padding: "8px 12px",
                                        borderRadius: 4,
                                        border: "1px solid #e0e0e0",
                                        resize: "vertical",
                                    }}
                                />
                            </div>
                        </div>
                    </DialogContent>
                    <DialogActions>
                        <Button variant="ghost" onClick={() => setIsOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="primary" onClick={() => setIsOpen(false)}>
                            Create Project
                        </Button>
                    </DialogActions>
                </Dialog>
            </>
        );
    },
};

// Nested dialogs
export const NestedDialogs: Story = {
    render: () => {
        const [isFirstOpen, setIsFirstOpen] = useState(false);
        const [isSecondOpen, setIsSecondOpen] = useState(false);

        return (
            <>
                <Button onClick={() => setIsFirstOpen(true)}>
                    Open First Dialog
                </Button>

                <Dialog
                    isOpen={isFirstOpen}
                    onClose={() => setIsFirstOpen(false)}
                    title="First Dialog"
                >
                    <DialogContent>
                        <Typography paragraph>
                            This is the first dialog. You can open another dialog on top of this one.
                        </Typography>
                        <Button variant="secondary" onClick={() => setIsSecondOpen(true)}>
                            Open Second Dialog
                        </Button>
                    </DialogContent>
                    <DialogActions>
                        <Button variant="primary" onClick={() => setIsFirstOpen(false)}>
                            Close
                        </Button>
                    </DialogActions>
                </Dialog>

                <Dialog
                    isOpen={isSecondOpen}
                    onClose={() => setIsSecondOpen(false)}
                    title="Second Dialog"
                    variant="success"
                    size="sm"
                >
                    <DialogContent>
                        <Typography>
                            This dialog is stacked on top of the first one.
                        </Typography>
                    </DialogContent>
                    <DialogActions>
                        <Button variant="primary" onClick={() => setIsSecondOpen(false)}>
                            Close
                        </Button>
                    </DialogActions>
                </Dialog>
            </>
        );
    },
};

// No background blur
export const NoBackgroundBlur: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button onClick={() => setIsOpen(true)}>
                    Open Dialog (No Background Blur)
                </Button>
                <Dialog
                    isOpen={isOpen}
                    onClose={() => setIsOpen(false)}
                    title="No Background Blur"
                    enableBackgroundBlur={false}
                >
                    <DialogContent>
                        <Typography>
                            This dialog has background blur disabled. Notice how the background
                            behind the dialog is completely transparent - no dimming and no blur.
                            This creates a much lighter visual effect.
                        </Typography>
                    </DialogContent>
                    <DialogActions>
                        <Button variant="primary" onClick={() => setIsOpen(false)}>
                            Close
                        </Button>
                    </DialogActions>
                </Dialog>
            </>
        );
    },
};

// Draggable dialog
export const DraggableDialog: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button onClick={() => setIsOpen(true)}>
                    Open Draggable Dialog
                </Button>
                <Dialog
                    isOpen={isOpen}
                    onClose={() => setIsOpen(false)}
                    title="Drag Me Around!"
                    draggable={true}
                    size="md"
                >
                    <DialogContent>
                        <Typography paragraph>
                            This dialog can be dragged around by clicking and holding the title bar.
                            Try grabbing the title area and moving the dialog to different positions.
                        </Typography>
                        <Typography paragraph>
                            The dialog is constrained to stay within the viewport bounds, so it won't
                            disappear off-screen when you drag it.
                        </Typography>
                        <Typography>
                            Notice how the cursor changes to a move cursor when hovering over the title bar.
                        </Typography>
                    </DialogContent>
                    <DialogActions>
                        <Button variant="ghost" onClick={() => setIsOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="primary" onClick={() => setIsOpen(false)}>
                            Save Position
                        </Button>
                    </DialogActions>
                </Dialog>
            </>
        );
    },
};

// Anchored dialog with arrow pointing to element
export const AnchoredDialog: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
        const [placement, setPlacement] = useState<"top" | "bottom" | "left" | "right" | "auto">("auto");

        const handleOpen = (event: React.MouseEvent<HTMLElement>) => {
            setAnchorEl(event.currentTarget);
            setIsOpen(true);
        };

        const handleClose = () => {
            setIsOpen(false);
            setAnchorEl(null);
        };

        return (
            <div style={{ padding: "40px", minHeight: "80vh", display: "flex", flexDirection: "column", gap: "20px" }}>
                <Typography variant="h6">Click any button to anchor the dialog to it:</Typography>
                
                <div style={{ display: "flex", gap: "20px", flexWrap: "wrap", alignItems: "flex-start" }}>
                    <Button 
                        variant="primary" 
                        onClick={handleOpen}
                        style={{ alignSelf: "flex-start" }}
                    >
                        Top Area Button
                    </Button>
                    
                    <Button 
                        variant="secondary" 
                        onClick={handleOpen}
                        style={{ alignSelf: "flex-start" }}
                    >
                        Another Button
                    </Button>

                    <Button 
                        variant="primary" 
                        onClick={handleOpen}
                        style={{ alignSelf: "flex-start" }}
                    >
                        Primary Button
                    </Button>
                </div>

                <div style={{ margin: "60px 0", display: "flex", justifyContent: "center" }}>
                    <Button 
                        variant="danger" 
                        onClick={handleOpen}
                        size="lg"
                    >
                        Center Button (Large)
                    </Button>
                </div>

                <div style={{ display: "flex", gap: "20px", justifyContent: "flex-end", marginTop: "40px" }}>
                    <Button 
                        variant="outline" 
                        onClick={handleOpen}
                    >
                        Bottom Right
                    </Button>
                    
                    <Button 
                        variant="ghost" 
                        onClick={handleOpen}
                    >
                        Far Right
                    </Button>
                </div>

                <div style={{ marginTop: "20px" }}>
                    <Typography variant="body2" style={{ marginBottom: "10px" }}>
                        Force placement (for testing):
                    </Typography>
                    <div style={{ display: "flex", gap: "10px" }}>
                        {(["auto", "top", "bottom", "left", "right"] as const).map((p) => (
                            <Button
                                key={p}
                                variant={placement === p ? "primary" : "outline"}
                                size="sm"
                                onClick={() => setPlacement(p)}
                            >
                                {p}
                            </Button>
                        ))}
                    </div>
                </div>

                <Dialog
                    isOpen={isOpen}
                    onClose={handleClose}
                    title="Anchored Dialog"
                    anchorEl={anchorEl}
                    anchorPlacement={placement}
                    highlightAnchor={true}
                    size="sm"
                >
                    <DialogContent>
                        <Typography paragraph>
                            This dialog is anchored to the button you clicked! Notice:
                        </Typography>
                        <ul style={{ marginLeft: "20px", marginBottom: "16px" }}>
                            <li>The arrow points to the button</li>
                            <li>The button is highlighted with a pulsing outline</li>
                            <li>The dialog automatically positions itself to avoid viewport edges</li>
                            <li>Dragging is disabled when anchored</li>
                        </ul>
                        <Typography>
                            Current placement: <strong>{placement === "auto" ? "auto (calculated)" : placement}</strong>
                        </Typography>
                    </DialogContent>
                    <DialogActions>
                        <Button variant="ghost" onClick={handleClose}>
                            Cancel
                        </Button>
                        <Button variant="primary" onClick={handleClose}>
                            Got it!
                        </Button>
                    </DialogActions>
                </Dialog>
            </div>
        );
    },
};
