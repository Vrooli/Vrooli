import { Typography } from "@mui/material";
import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { useIsMobile, useWindowDimensions } from "../../../hooks/useIsMobile.js";
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
        const [contentType, setContentType] = useState<"short" | "medium" | "long">("medium");
        const sizes = ["sm", "md", "lg", "xl", "full"] as const;
        const isMobile = useIsMobile();
        const windowSize = useWindowDimensions();

        // Content variations for testing
        const contentVariations = {
            short: {
                title: "Short Content",
                content: (
                    <Typography>
                        This is a short dialog with minimal content to test how mobile full screen behaves with small content.
                    </Typography>
                ),
            },
            medium: {
                title: "Medium Content", 
                content: (
                    <div>
                        <Typography paragraph>
                            This dialog has a moderate amount of content to test the mobile full screen behavior.
                        </Typography>
                        <Typography paragraph>
                            Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor 
                            incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud 
                            exercitation ullamco laboris.
                        </Typography>
                        <Typography>
                            Notice how the dialog should only be as tall as needed on mobile, sitting at the bottom.
                        </Typography>
                    </div>
                ),
            },
            long: {
                title: "Long Content",
                content: (
                    <div>
                        {Array.from({ length: 8 }, (_, i) => (
                            <Typography key={i} paragraph>
                                Section {i + 1}: Lorem ipsum dolor sit amet, consectetur adipiscing elit. 
                                Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad 
                                minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea 
                                commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit 
                                esse cillum dolore eu fugiat nulla pariatur.
                            </Typography>
                        ))}
                        <Typography>
                            This long content should trigger scrolling when the dialog hits its maximum height 
                            on mobile (calc(100vh - 2rem)).
                        </Typography>
                    </div>
                ),
            },
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", gap: "24px" }}>
                {/* Device Detection Status */}
                <div style={{
                    padding: "12px 16px",
                    backgroundColor: isMobile ? "#e3f2fd" : "#f3e5f5",
                    borderRadius: "8px",
                    border: `2px solid ${isMobile ? "#2196f3" : "#9c27b0"}`,
                    textAlign: "center"
                }}>
                    <Typography variant="body1" style={{ fontWeight: "bold", marginBottom: "4px" }}>
                        📱 Current Viewport: {windowSize.width}×{windowSize.height}px {isMobile ? "(Mobile)" : "(Desktop)"}
                    </Typography>
                    <Typography variant="body2" style={{ color: isMobile ? "#1976d2" : "#7b1fa2" }}>
                        {isMobile 
                            ? "🔄 Full size dialogs will slide up from bottom with rounded top corners"
                            : "📺 Full size dialogs will behave like XL size (centered)"}
                    </Typography>
                    <Typography variant="caption" style={{ display: "block", marginTop: "4px", opacity: 0.8 }}>
                        Breakpoint: {isMobile ? "< 768px" : "≥ 768px"} • Resize window to test responsive behavior
                    </Typography>
                </div>

                {/* Content Type Selector */}
                <div style={{
                    padding: "16px",
                    backgroundColor: "#f5f5f5",
                    borderRadius: "8px",
                    border: "1px solid #ddd"
                }}>
                    <Typography variant="body2" style={{ marginBottom: "12px", fontWeight: "bold" }}>
                        📝 Content Amount (especially useful for testing mobile "full" size):
                    </Typography>
                    <div style={{ display: "flex", gap: "8px", flexWrap: "wrap" }}>
                        {(["short", "medium", "long"] as const).map((type) => (
                            <Button
                                key={type}
                                variant={contentType === type ? "primary" : "outline"}
                                size="sm"
                                onClick={() => setContentType(type)}
                            >
                                {type.charAt(0).toUpperCase() + type.slice(1)} Content
                            </Button>
                        ))}
                    </div>
                    <Typography variant="caption" style={{ display: "block", marginTop: "8px", opacity: 0.7 }}>
                        Test how mobile dialogs size to content: Short = minimal height, Long = scrollable
                    </Typography>
                </div>

                {/* Size Buttons */}
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
                </div>

                {/* Dialog Components */}
                {sizes.map((size) => {
                    const selectedContent = contentVariations[contentType];
                    return (
                        <Dialog
                            key={`${size}-${contentType}`} // Include contentType in key to force re-render
                            isOpen={openDialog === size}
                            onClose={() => setOpenDialog(null)}
                            title={`${size.toUpperCase()} Size - ${selectedContent.title}`}
                            size={size}
                        >
                            <DialogContent>
                                {/* Size info */}
                                <Typography sx={{ mb: 2, p: 1.5, bgcolor: "action.hover", borderRadius: 1, fontSize: "0.875rem" }}>
                                    <strong>Size:</strong> {size} • <strong>Content:</strong> {contentType}
                                    {size === "full" && (
                                        <>
                                            <br />
                                            <strong>Mobile behavior:</strong> {isMobile ? "Slide-up from bottom" : "Standard XL-size"}
                                        </>
                                    )}
                                </Typography>

                                {/* Selected content */}
                                {selectedContent.content}

                                {/* Additional info for full size dialogs */}
                                {size === "full" && (
                                    <div style={{ marginTop: "24px", padding: "16px", backgroundColor: isMobile ? "#e3f2fd" : "#f3e5f5", borderRadius: "8px" }}>
                                        <Typography sx={{ mb: 1, fontWeight: "bold" }}>
                                            📱 Full Size Behavior Details:
                                        </Typography>
                                        <Typography sx={{ mb: 1, fontSize: "0.875rem" }}>
                                            • <strong>Desktop:</strong> Displays as XL size (centered, max-width)
                                        </Typography>
                                        <Typography sx={{ mb: 1, fontSize: "0.875rem" }}>
                                            • <strong>Mobile:</strong> Slides up from bottom, full width, rounded top corners only
                                        </Typography>
                                        <Typography sx={{ mb: 1, fontSize: "0.875rem" }}>
                                            • <strong>Height:</strong> Only as tall as content needs (up to max)
                                        </Typography>
                                        <Typography sx={{ mt: 1.5, p: 1.5, bgcolor: "background.paper", borderRadius: 1, fontSize: "0.875rem" }}>
                                            <strong>Current:</strong> {windowSize.width}×{windowSize.height}px {isMobile ? "(Mobile)" : "(Desktop)"}
                                            <br />
                                            <strong>Effective:</strong> {isMobile ? "Mobile slide-up behavior" : "Desktop XL-size behavior"}
                                        </Typography>
                                    </div>
                                )}
                            </DialogContent>
                            <DialogActions>
                                <Button variant="ghost" onClick={() => setOpenDialog(null)}>
                                    Cancel
                                </Button>
                                <Button variant="primary" onClick={() => setOpenDialog(null)}>
                                    Close
                                </Button>
                            </DialogActions>
                        </Dialog>
                    );
                })}
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
        const centerRef = React.useRef<HTMLDivElement>(null);
        const scrollContainerRef = React.useRef<HTMLDivElement>(null);

        const handleOpen = (event: React.MouseEvent<HTMLElement>) => {
            setAnchorEl(event.currentTarget);
            setIsOpen(true);
        };

        const handleClose = () => {
            setIsOpen(false);
            setAnchorEl(null);
        };

        // Scroll to center on mount
        React.useEffect(() => {
            if (centerRef.current && scrollContainerRef.current) {
                const centerRect = centerRef.current.getBoundingClientRect();
                const containerRect = scrollContainerRef.current.getBoundingClientRect();
                
                scrollContainerRef.current.scrollTo({
                    left: (scrollContainerRef.current.scrollWidth - containerRect.width) / 2,
                    top: (scrollContainerRef.current.scrollHeight - containerRect.height) / 2,
                    behavior: "auto"
                });
            }
        }, []);

        return (
            <div 
                ref={scrollContainerRef}
                style={{ 
                    position: "fixed",
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    overflow: "auto",
                    backgroundColor: "var(--background-default)"
                }}
            >
                <div style={{ 
                    minHeight: "300vh", 
                    width: "150vw", 
                    padding: "40px",
                    position: "relative"
                }}>
                <div style={{ maxWidth: "800px", margin: "0 auto", padding: "20px" }}>
                    <Typography variant="h4" style={{ marginBottom: "20px", textAlign: "center" }}>
                        🎯 Anchored Dialog Test Arena
                    </Typography>
                    <Typography variant="body1" style={{ marginBottom: "40px", textAlign: "center" }}>
                        Scroll around and click buttons to see dynamic dialog positioning!
                    </Typography>
                </div>

                {/* Top Section */}
                <div style={{ 
                    marginBottom: "100vh", 
                    display: "flex", 
                    flexDirection: "column", 
                    alignItems: "center", 
                    gap: "40px",
                    padding: "40px"
                }}>
                    <Typography variant="h5">🔝 Top Section</Typography>
                    <div style={{ display: "flex", gap: "20px", flexWrap: "wrap", justifyContent: "center" }}>
                        <Button variant="primary" onClick={handleOpen}>Top Left</Button>
                        <Button variant="secondary" onClick={handleOpen}>Top Center</Button>
                        <Button variant="danger" onClick={handleOpen}>Top Right</Button>
                    </div>
                    <Typography variant="body2" style={{ textAlign: "center", maxWidth: "600px" }}>
                        When you click these buttons, the dialog should appear below them since there's lots of space below.
                    </Typography>
                </div>

                {/* Left Section */}
                <div style={{ 
                    position: "absolute",
                    left: "40px",
                    top: "50%",
                    transform: "translateY(-50%)",
                    display: "flex",
                    flexDirection: "column",
                    gap: "20px",
                    alignItems: "center"
                }}>
                    <Typography variant="h6">⬅️ Left Edge</Typography>
                    <Button variant="outline" onClick={handleOpen}>Left Top</Button>
                    <Button variant="ghost" onClick={handleOpen}>Left Middle</Button>
                    <Button variant="primary" onClick={handleOpen}>Left Bottom</Button>
                </div>

                {/* Right Section */}
                <div style={{ 
                    position: "absolute",
                    right: "40px",
                    top: "50%",
                    transform: "translateY(-50%)",
                    display: "flex",
                    flexDirection: "column",
                    gap: "20px",
                    alignItems: "center"
                }}>
                    <Typography variant="h6">➡️ Right Edge</Typography>
                    <Button variant="space" onClick={handleOpen}>Right Top</Button>
                    <Button variant="neon" onClick={handleOpen}>Right Middle</Button>
                    <Button variant="secondary" onClick={handleOpen}>Right Bottom</Button>
                </div>

                {/* Center Section - This is where we'll scroll to initially */}
                <div 
                    ref={centerRef}
                    style={{ 
                        position: "absolute",
                        left: "50%",
                        top: "50%",
                        transform: "translate(-50%, -50%)",
                        display: "flex",
                        flexDirection: "column",
                        alignItems: "center",
                        gap: "30px",
                        padding: "60px",
                        backgroundColor: "var(--background-paper)",
                        borderRadius: "20px",
                        border: "3px solid #2196f3",
                        boxShadow: "0 0 30px rgba(33, 150, 243, 0.3)"
                    }}
                >
                    <Typography variant="h4" style={{ textAlign: "center" }}>
                        🎯 Center Stage
                    </Typography>
                    <Typography variant="body1" style={{ textAlign: "center", maxWidth: "400px" }}>
                        You start here! Try clicking buttons and then scrolling around to see how the dialog repositions itself automatically.
                    </Typography>
                    
                    <div style={{ display: "flex", gap: "15px", flexWrap: "wrap", justifyContent: "center" }}>
                        <Button variant="primary" onClick={handleOpen} size="lg">
                            🎯 Center Target
                        </Button>
                        <Button variant="danger" onClick={handleOpen}>
                            Emergency
                        </Button>
                        <Button variant="secondary" onClick={handleOpen}>
                            Secondary
                        </Button>
                    </div>

                    <div style={{ marginTop: "20px" }}>
                        <Typography variant="body2" style={{ marginBottom: "10px", textAlign: "center" }}>
                            Force placement (overrides auto):
                        </Typography>
                        <div style={{ display: "flex", gap: "8px", justifyContent: "center" }}>
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
                </div>

                {/* Bottom Section */}
                <div style={{ 
                    position: "absolute",
                    bottom: "40px",
                    left: "50%",
                    transform: "translateX(-50%)",
                    display: "flex",
                    flexDirection: "column",
                    alignItems: "center",
                    gap: "20px",
                    padding: "40px"
                }}>
                    <Typography variant="h5">🔻 Bottom Section</Typography>
                    <div style={{ display: "flex", gap: "20px", flexWrap: "wrap", justifyContent: "center" }}>
                        <Button variant="outline" onClick={handleOpen}>Bottom Left</Button>
                        <Button variant="ghost" onClick={handleOpen}>Bottom Center</Button>
                        <Button variant="danger" onClick={handleOpen}>Bottom Right</Button>
                    </div>
                    <Typography variant="body2" style={{ textAlign: "center", maxWidth: "600px" }}>
                        These buttons should show dialogs above them since there's no space below.
                    </Typography>
                </div>

                {/* Corner Buttons for Edge Cases */}
                <Button 
                    variant="primary" 
                    onClick={handleOpen}
                    style={{ position: "absolute", top: "20px", left: "20px" }}
                >
                    Top-Left Corner
                </Button>
                
                <Button 
                    variant="secondary" 
                    onClick={handleOpen}
                    style={{ position: "absolute", top: "20px", right: "20px" }}
                >
                    Top-Right Corner
                </Button>
                
                <Button 
                    variant="danger" 
                    onClick={handleOpen}
                    style={{ position: "absolute", bottom: "20px", left: "20px" }}
                >
                    Bottom-Left Corner
                </Button>
                
                <Button 
                    variant="outline" 
                    onClick={handleOpen}
                    style={{ position: "absolute", bottom: "20px", right: "20px" }}
                >
                    Bottom-Right Corner
                </Button>

                <Dialog
                    isOpen={isOpen}
                    onClose={handleClose}
                    title="🎯 Smart Anchored Dialog"
                    anchorEl={anchorEl}
                    anchorPlacement={placement}
                    highlightAnchor={true}
                    size="sm"
                >
                    <DialogContent>
                        <Typography paragraph>
                            🎉 This dialog intelligently positions itself! 
                        </Typography>
                        <ul style={{ marginLeft: "20px", marginBottom: "16px" }}>
                            <li>📍 Arrow points to the clicked button</li>
                            <li>✨ Button gets a pulsing highlight</li>
                            <li>🧠 Auto-calculates best position based on available space</li>
                            <li>📜 Updates position as you scroll around</li>
                            <li>🚫 Dragging disabled when anchored</li>
                        </ul>
                        <Typography>
                            Current placement: <strong>{placement === "auto" ? "auto (smart)" : placement}</strong>
                        </Typography>
                        <Typography variant="body2" style={{ marginTop: "16px", fontStyle: "italic" }}>
                            💡 Try scrolling around while this dialog is open to see it reposition!
                        </Typography>
                    </DialogContent>
                    <DialogActions>
                        <Button variant="ghost" onClick={handleClose}>
                            Cancel
                        </Button>
                        <Button variant="primary" onClick={handleClose}>
                            Awesome! 🎯
                        </Button>
                    </DialogActions>
                </Dialog>
                </div>
            </div>
        );
    },
};
