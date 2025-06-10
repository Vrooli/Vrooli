import type { Meta, StoryObj } from "@storybook/react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { useState } from "react";
import { IconCommon } from "../../icons/Icons.js";
import { TailwindButton, Button } from "./TailwindButton.js";
import type { ButtonVariant, ButtonSize } from "./TailwindButton.js";

const meta: Meta<typeof TailwindButton> = {
    title: "Components/Buttons/TailwindButton",
    component: TailwindButton,
    parameters: {
        layout: "centered",
    },
    argTypes: {
        variant: {
            control: { type: "select" },
            options: ["primary", "secondary", "outline", "ghost", "danger"],
        },
        size: {
            control: { type: "select" },
            options: ["sm", "md", "lg", "icon"],
        },
        isLoading: {
            control: { type: "boolean" },
        },
        disabled: {
            control: { type: "boolean" },
        },
        fullWidth: {
            control: { type: "boolean" },
        },
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Basic stories
export const Default: Story = {
    args: {
        children: "Click Me",
        variant: "primary",
        size: "md",
    },
};

export const Loading: Story = {
    args: {
        ...Default.args,
        isLoading: true,
    },
};

export const Disabled: Story = {
    args: {
        ...Default.args,
        disabled: true,
    },
};

export const WithStartIcon: Story = {
    args: {
        ...Default.args,
        startIcon: <IconCommon name="Add" />,
        children: "Add Item",
    },
};

export const WithEndIcon: Story = {
    args: {
        ...Default.args,
        endIcon: <IconCommon name="Send" />,
        children: "Send",
    },
};

export const IconOnly: Story = {
    args: {
        size: "icon",
        variant: "outline",
        children: <IconCommon name="Delete" />,
    },
};

// Showcase all variants and sizes
export const AllVariantsAndSizes = () => {
    const variants: ButtonVariant[] = ["primary", "secondary", "outline", "ghost", "danger"];
    const sizes: ButtonSize[] = ["sm", "md", "lg"];

    return (
        <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 4 }}>
            <Typography variant="h5">All Button Variants and Sizes</Typography>
            
            {variants.map(variant => (
                <Box key={variant} sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                    <Typography variant="h6" sx={{ textTransform: "capitalize" }}>
                        {variant} Variant
                    </Typography>
                    
                    <Box sx={{ display: "flex", gap: 2, alignItems: "center", flexWrap: "wrap" }}>
                        {sizes.map(size => (
                            <TailwindButton
                                key={`${variant}-${size}`}
                                variant={variant}
                                size={size}
                            >
                                {size.toUpperCase()} Button
                            </TailwindButton>
                        ))}
                        <TailwindButton
                            variant={variant}
                            size="icon"
                        >
                            <IconCommon name="Add" />
                        </TailwindButton>
                    </Box>
                </Box>
            ))}
        </Box>
    );
};

// Interactive loading demo
export const InteractiveLoadingDemo = () => {
    const [loadingStates, setLoadingStates] = useState<Record<string, boolean>>({});

    const handleClick = (key: string) => {
        setLoadingStates(prev => ({ ...prev, [key]: true }));
        setTimeout(() => {
            setLoadingStates(prev => ({ ...prev, [key]: false }));
        }, 2000);
    };

    const variants: ButtonVariant[] = ["primary", "secondary", "outline", "ghost", "danger"];

    return (
        <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3, alignItems: "center" }}>
            <Typography variant="h5">Interactive Loading Demo</Typography>
            <Typography variant="body2" color="text.secondary">
                Click any button to see the loading state
            </Typography>
            
            <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap", justifyContent: "center" }}>
                {variants.map(variant => (
                    <TailwindButton
                        key={variant}
                        variant={variant}
                        isLoading={loadingStates[variant] || false}
                        onClick={() => handleClick(variant)}
                        startIcon={!loadingStates[variant] && <IconCommon name="Send" />}
                    >
                        {variant.charAt(0).toUpperCase() + variant.slice(1)}
                    </TailwindButton>
                ))}
            </Box>
        </Box>
    );
};

// Component factory demo
export const ComponentFactoryDemo = () => {
    return (
        <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3 }}>
            <Typography variant="h5">Component Factory Pattern</Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Use pre-configured button components for consistency
            </Typography>
            
            <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap" }}>
                <Button.Primary startIcon={<IconCommon name="Add" />}>
                    Primary Action
                </Button.Primary>
                
                <Button.Secondary>
                    Secondary Action
                </Button.Secondary>
                
                <Button.Outline startIcon={<IconCommon name="Send" />}>
                    Send Message
                </Button.Outline>
                
                <Button.Ghost>
                    Cancel
                </Button.Ghost>
                
                <Button.Danger startIcon={<IconCommon name="Delete" />}>
                    Delete Item
                </Button.Danger>
            </Box>
        </Box>
    );
};

// Accessibility demo
export const AccessibilityDemo = () => {
    const [isLoading, setIsLoading] = useState(false);
    const [isSuccess, setIsSuccess] = useState(false);

    const handleSubmit = () => {
        setIsLoading(true);
        setIsSuccess(false);
        
        setTimeout(() => {
            setIsLoading(false);
            setIsSuccess(true);
            setTimeout(() => setIsSuccess(false), 2000);
        }, 2000);
    };

    return (
        <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3, maxWidth: 400 }}>
            <Typography variant="h5">Accessibility Features</Typography>
            
            <Box sx={{ p: 3, bgcolor: "background.paper", borderRadius: 2 }}>
                <Typography variant="body2" sx={{ mb: 2 }}>
                    The button includes proper ARIA attributes:
                </Typography>
                <ul>
                    <li>aria-busy when loading</li>
                    <li>aria-disabled when disabled</li>
                    <li>Keyboard navigation support</li>
                    <li>Focus ring for keyboard users</li>
                </ul>
            </Box>
            
            <Box sx={{ display: "flex", gap: 2, alignItems: "center" }}>
                <TailwindButton
                    variant="primary"
                    onClick={handleSubmit}
                    isLoading={isLoading}
                    disabled={isSuccess}
                >
                    {isLoading ? "Submitting..." : isSuccess ? "Success!" : "Submit Form"}
                </TailwindButton>
                
                {isSuccess && (
                    <Typography variant="body2" color="success.main">
                        ✓ Form submitted successfully
                    </Typography>
                )}
            </Box>
            
            <Typography variant="caption" color="text.secondary">
                Try navigating with Tab and activating with Space/Enter
            </Typography>
        </Box>
    );
};