import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { IconCommon } from "../../icons/Icons.js";
import { DotsLoader, GradientRingLoader } from "../indicators/CircularProgress.js";
import type { ButtonVariant } from "./TailwindButton.js";
import { TailwindButton } from "./TailwindButton.js";

const meta: Meta<typeof TailwindButton> = {
    title: "Components/Buttons/TailwindButton Enhanced",
    component: TailwindButton,
    parameters: {
        layout: "centered",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Showcase the new loading animations
export const EnhancedLoadingStates = () => {
    const [loadingStates, setLoadingStates] = useState<Record<string, boolean>>({});
    const variants: ButtonVariant[] = ["primary", "secondary", "outline", "ghost", "danger"];

    const handleClick = (key: string) => {
        setLoadingStates(prev => ({ ...prev, [key]: true }));
        setTimeout(() => {
            setLoadingStates(prev => ({ ...prev, [key]: false }));
        }, 3000);
    };

    return (
        <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 4 }}>
            <Typography variant="h5">Enhanced Loading Animations</Typography>

            <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                <Typography variant="h6">Click to see loading states:</Typography>
                {variants.map(variant => (
                    <TailwindButton
                        key={variant}
                        variant={variant}
                        isLoading={loadingStates[variant] || false}
                        onClick={() => handleClick(variant)}
                        startIcon={!loadingStates[variant] && <IconCommon name="Save" />}
                    >
                        {loadingStates[variant] ? "Processing..." : `${variant} Button`}
                    </TailwindButton>
                ))}
            </Box>

            <Typography variant="body2" color="text.secondary">
                Our custom CircularProgress component provides smooth animations and better theme integration
            </Typography>
        </Box>
    );
};

// Alternative loading indicators
export const AlternativeLoadingIndicators = () => {
    const [loadingType, setLoadingType] = useState<"circular" | "orbital" | "dots" | "gradient">("orbital");
    const [isLoading, setIsLoading] = useState(false);

    const handleSubmit = () => {
        setIsLoading(true);
        setTimeout(() => setIsLoading(false), 2000);
    };

    return (
        <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3, maxWidth: 600 }}>
            <Typography variant="h5">Alternative Loading Styles</Typography>

            <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap" }}>
                <label>
                    <input
                        type="radio"
                        value="circular"
                        checked={loadingType === "circular"}
                        onChange={(e) => setLoadingType(e.target.value as any)}
                    />
                    Circular
                </label>
                <label>
                    <input
                        type="radio"
                        value="orbital"
                        checked={loadingType === "orbital"}
                        onChange={(e) => setLoadingType(e.target.value as any)}
                    />
                    Orbital (Space Theme)
                </label>
                <label>
                    <input
                        type="radio"
                        value="dots"
                        checked={loadingType === "dots"}
                        onChange={(e) => setLoadingType(e.target.value as any)}
                    />
                    Dots
                </label>
                <label>
                    <input
                        type="radio"
                        value="gradient"
                        checked={loadingType === "gradient"}
                        onChange={(e) => setLoadingType(e.target.value as any)}
                    />
                    Gradient
                </label>
            </Box>

            <Box sx={{ display: "flex", gap: 2, alignItems: "center", flexWrap: "wrap" }}>
                <TailwindButton
                    variant="primary"
                    onClick={handleSubmit}
                    isLoading={isLoading && loadingType === "circular"}
                    loadingIndicator="circular"
                >
                    Submit (Circular)
                </TailwindButton>

                <TailwindButton
                    variant="primary"
                    onClick={handleSubmit}
                    isLoading={isLoading && loadingType === "orbital"}
                    loadingIndicator="orbital"
                >
                    Submit (Orbital)
                </TailwindButton>

                <button
                    onClick={handleSubmit}
                    disabled={isLoading}
                    className="tw-inline-flex tw-items-center tw-gap-2 tw-bg-secondary-main tw-text-white tw-px-4 tw-py-2 tw-rounded tw-font-medium disabled:tw-opacity-50"
                >
                    {isLoading && loadingType === "dots" && <DotsLoader size={20} />}
                    Submit (Dots)
                </button>

                <button
                    onClick={handleSubmit}
                    disabled={isLoading}
                    className="tw-inline-flex tw-items-center tw-gap-2 tw-bg-secondary-main tw-text-white tw-px-4 tw-py-2 tw-rounded tw-font-medium disabled:tw-opacity-50"
                >
                    {isLoading && loadingType === "gradient" && <GradientRingLoader size={20} />}
                    Submit (Gradient)
                </button>
            </Box>
        </Box>
    );
};

// Size comparison
export const LoadingSizeComparison = () => {
    const [isLoading, setIsLoading] = useState(true);

    return (
        <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3 }}>
            <Box sx={{ display: "flex", gap: 2, alignItems: "center" }}>
                <Typography variant="h6">Loading spinner scales with button size:</Typography>
                <button
                    onClick={() => setIsLoading(!isLoading)}
                    className="tw-text-secondary-main tw-underline"
                >
                    Toggle Loading
                </button>
            </Box>

            <Box sx={{ display: "flex", gap: 2, alignItems: "center" }}>
                <TailwindButton size="sm" variant="primary" isLoading={isLoading}>
                    Small
                </TailwindButton>
                <TailwindButton size="md" variant="primary" isLoading={isLoading}>
                    Medium
                </TailwindButton>
                <TailwindButton size="lg" variant="primary" isLoading={isLoading}>
                    Large
                </TailwindButton>
                <TailwindButton size="icon" variant="primary" isLoading={isLoading}>
                    <IconCommon name="Save" />
                </TailwindButton>
            </Box>
        </Box>
    );
};

// Performance demo
export const PerformanceDemo = () => {
    const [loadingStates, setLoadingStates] = useState<Record<number, boolean>>({});

    const toggleAll = () => {
        const newStates: Record<number, boolean> = {};
        for (let i = 0; i < 20; i++) {
            newStates[i] = true;
        }
        setLoadingStates(newStates);

        setTimeout(() => {
            setLoadingStates({});
        }, 3000);
    };

    return (
        <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3 }}>
            <Typography variant="h5">Performance Test: 20 Loading Buttons</Typography>
            <Typography variant="body2" color="text.secondary">
                All animations use CSS only - no JavaScript animations for optimal performance
            </Typography>

            <button
                onClick={toggleAll}
                className="tw-self-start tw-bg-gray-800 tw-text-white tw-px-4 tw-py-2 tw-rounded"
            >
                Start All Loading States
            </button>

            <Box sx={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(150px, 1fr))", gap: 2 }}>
                {Array.from({ length: 20 }).map((_, i) => (
                    <TailwindButton
                        key={i}
                        variant={["primary", "secondary", "outline", "ghost", "danger"][i % 5] as ButtonVariant}
                        size="sm"
                        isLoading={loadingStates[i] || false}
                        onClick={() => {
                            setLoadingStates(prev => ({ ...prev, [i]: true }));
                            setTimeout(() => {
                                setLoadingStates(prev => ({ ...prev, [i]: false }));
                            }, 2000);
                        }}
                    >
                        Button {i + 1}
                    </TailwindButton>
                ))}
            </Box>
        </Box>
    );
};