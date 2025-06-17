import { Box, Typography } from "@mui/material";
import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions } from "./Dialog.js";
import { Button } from "../../buttons/Button.js";

const meta: Meta<typeof Dialog> = {
    title: "Components/Dialogs/Dialog (Tailwind)",
    component: Dialog,
    parameters: {
        layout: "centered",
        backgrounds: {
            default: 'transparent',
        },
    },
    tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Default dialog
export const Default: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button onClick={() => setIsOpen(true)}>
                    Open Default Dialog
                </Button>
                <Dialog
                    isOpen={isOpen}
                    onClose={() => setIsOpen(false)}
                    title="Default Dialog"
                >
                    <DialogContent>
                        <Typography>
                            This is a Tailwind-based dialog component that mimics the MUI Dialog functionality.
                        </Typography>
                    </DialogContent>
                    <DialogActions>
                        <Button variant="ghost" onClick={() => setIsOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="primary" onClick={() => setIsOpen(false)}>
                            Confirm
                        </Button>
                    </DialogActions>
                </Dialog>
            </>
        );
    },
};

// Sizes demonstration
export const Sizes: Story = {
    render: () => {
        const [openDialog, setOpenDialog] = useState<string | null>(null);
        const sizes = ["sm", "md", "lg", "xl", "full"] as const;

        return (
            <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap", backgroundColor: "transparent" }}>
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
            </Box>
        );
    },
};

// Variants demonstration (button styling)
export const Variants: Story = {
    render: () => {
        const [openDialog, setOpenDialog] = useState<string | null>(null);
        const variants = ["default", "danger", "success", "space", "neon"] as const;

        return (
            <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap", backgroundColor: "transparent" }}>
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
            </Box>
        );
    },
};

// Positions demonstration
export const Positions: Story = {
    render: () => {
        const [openDialog, setOpenDialog] = useState<string | null>(null);
        const positions = ["center", "top", "bottom", "left", "right"] as const;

        return (
            <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap", backgroundColor: "transparent" }}>
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
            </Box>
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
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                            <Box>
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
                            </Box>
                            <Box>
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
                            </Box>
                        </Box>
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

// Space themed dialog (special variant)
export const SpaceThemed: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button variant="space" onClick={() => setIsOpen(true)}>
                    Open Space Dialog
                </Button>
                <Dialog
                    isOpen={isOpen}
                    onClose={() => setIsOpen(false)}
                    title="Welcome to Space"
                    variant="space"
                    size="lg"
                >
                    <DialogContent>
                        <Typography sx={{ color: "white", mb: 2 }}>
                            Experience the cosmic theme with animated stars and nebula effects.
                        </Typography>
                        <Typography sx={{ color: "rgba(255, 255, 255, 0.8)" }}>
                            The space variant is special - it applies both themed styling to the dialog 
                            itself AND affects the action buttons. This is perfect for space-themed 
                            applications or when you want to create a futuristic feel.
                        </Typography>
                    </DialogContent>
                    <DialogActions variant="space">
                        <Button variant="ghost" onClick={() => setIsOpen(false)}>
                            Return to Earth
                        </Button>
                        <Button variant="space" onClick={() => setIsOpen(false)}>
                            Launch Mission
                        </Button>
                    </DialogActions>
                </Dialog>
            </>
        );
    },
};

// Neon themed dialog (special variant)
export const NeonThemed: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button variant="neon" onClick={() => setIsOpen(true)}>
                    Open Neon Dialog
                </Button>
                <Dialog
                    isOpen={isOpen}
                    onClose={() => setIsOpen(false)}
                    title="Neon Glow"
                    variant="neon"
                    size="md"
                >
                    <DialogContent>
                        <Typography sx={{ color: "#00ff7f", mb: 2 }}>
                            Welcome to the neon-lit future!
                        </Typography>
                        <Typography sx={{ color: "rgba(255, 255, 255, 0.9)" }}>
                            Like the space variant, neon is special - it applies both visual effects 
                            to the dialog and themed styling to the action buttons. Perfect for 
                            cyberpunk or retro-futuristic themes.
                        </Typography>
                    </DialogContent>
                    <DialogActions variant="neon">
                        <Button variant="ghost" onClick={() => setIsOpen(false)}>
                            Power Off
                        </Button>
                        <Button variant="neon" onClick={() => setIsOpen(false)}>
                            Activate
                        </Button>
                    </DialogActions>
                </Dialog>
            </>
        );
    },
};