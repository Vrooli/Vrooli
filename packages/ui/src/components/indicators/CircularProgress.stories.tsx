import type { Meta, StoryObj } from "@storybook/react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { CircularProgress, DotsLoader, GradientRingLoader, OrbitalSpinner } from "./CircularProgress.js";

const meta: Meta<typeof CircularProgress> = {
    title: "Components/Indicators/CircularProgress",
    component: CircularProgress,
    parameters: {
        layout: "centered",
    },
    argTypes: {
        size: {
            control: { type: "range", min: 16, max: 64, step: 4 },
        },
        thickness: {
            control: { type: "range", min: 1, max: 10, step: 1 },
        },
        variant: {
            control: { type: "select" },
            options: ["primary", "secondary", "white", "current"],
        },
        dualRing: {
            control: { type: "boolean" },
        },
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Basic story
export const Default: Story = {
    args: {
        size: 24,
        thickness: 3,
        variant: "primary",
        dualRing: false,
    },
};

// Size variations
export const Sizes = () => (
    <Box sx={{ display: "flex", gap: 3, alignItems: "center" }}>
        <CircularProgress size={16} />
        <CircularProgress size={24} />
        <CircularProgress size={32} />
        <CircularProgress size={48} />
        <CircularProgress size={64} />
    </Box>
);

// Thickness variations
export const ThicknessVariations = () => (
    <Box sx={{ display: "flex", gap: 3, alignItems: "center" }}>
        <CircularProgress size={40} thickness={1} />
        <CircularProgress size={40} thickness={3} />
        <CircularProgress size={40} thickness={5} />
        <CircularProgress size={40} thickness={8} />
        <CircularProgress size={40} thickness={10} />
    </Box>
);

// Color variants
export const ColorVariants = () => (
    <Box sx={{ display: "flex", gap: 3, alignItems: "center", p: 4 }}>
        <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 1 }}>
            <CircularProgress variant="primary" />
            <Typography variant="caption">Primary</Typography>
        </Box>
        <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 1 }}>
            <CircularProgress variant="secondary" />
            <Typography variant="caption">Secondary</Typography>
        </Box>
        <Box sx={{ bgcolor: "grey.800", p: 2, borderRadius: 1, display: "flex", flexDirection: "column", alignItems: "center", gap: 1 }}>
            <CircularProgress variant="white" />
            <Typography variant="caption" color="white">White</Typography>
        </Box>
        <Box sx={{ color: "error.main", display: "flex", flexDirection: "column", alignItems: "center", gap: 1 }}>
            <CircularProgress variant="current" />
            <Typography variant="caption" color="error">Current Color</Typography>
        </Box>
    </Box>
);

// Dual ring spinner
export const DualRingSpinner = () => (
    <Box sx={{ display: "flex", gap: 4, alignItems: "center" }}>
        <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 2 }}>
            <CircularProgress size={40} dualRing />
            <Typography variant="caption">Dual Ring</Typography>
        </Box>
        <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 2 }}>
            <CircularProgress size={40} />
            <Typography variant="caption">Single Ring</Typography>
        </Box>
    </Box>
);

// Alternative loaders
export const AlternativeLoaders = () => (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 4, p: 4 }}>
        <Typography variant="h6">Alternative Loading Indicators</Typography>
        
        <Box sx={{ display: "flex", gap: 4, alignItems: "center", flexWrap: "wrap" }}>
            <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 2 }}>
                <DotsLoader size={24} />
                <Typography variant="caption">Dots Loader</Typography>
            </Box>
            
            <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 2 }}>
                <GradientRingLoader size={32} />
                <Typography variant="caption">Gradient Ring</Typography>
            </Box>
            
            <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 2 }}>
                <OrbitalSpinner size={40} />
                <Typography variant="caption">Orbital Spinner</Typography>
            </Box>
        </Box>
        
        <Typography variant="body2" color="text.secondary">
            These alternative loaders provide different visual styles for various use cases. The Orbital Spinner is perfect for Vrooli's space theme!
        </Typography>
    </Box>
);

// In-context examples
export const InContextExamples = () => (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 3, p: 4 }}>
        <Typography variant="h6">Loading States in Context</Typography>
        
        {/* Loading card */}
        <Box sx={{ p: 3, bgcolor: "background.paper", borderRadius: 2, display: "flex", alignItems: "center", gap: 2 }}>
            <CircularProgress size={20} />
            <Typography>Loading user data...</Typography>
        </Box>
        
        {/* Loading overlay */}
        <Box sx={{ position: "relative", p: 4, bgcolor: "grey.100", borderRadius: 2, minHeight: 100 }}>
            <Typography color="text.secondary">Content would be here</Typography>
            <Box sx={{ 
                position: "absolute", 
                inset: 0, 
                display: "flex", 
                alignItems: "center", 
                justifyContent: "center",
                bgcolor: "rgba(255, 255, 255, 0.8)",
                borderRadius: 2,
            }}>
                <CircularProgress size={32} />
            </Box>
        </Box>
        
        {/* Inline loading */}
        <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <Typography>Processing</Typography>
            <DotsLoader size={16} />
        </Box>
    </Box>
);

// Orbital Spinner showcase
export const OrbitalSpinnerShowcase = () => (
    <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 4 }}>
        <Typography variant="h5">Orbital Spinner - Vrooli Space Theme</Typography>
        
        <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
            <Typography variant="h6">Different Sizes</Typography>
            <Box sx={{ display: "flex", gap: 3, alignItems: "center" }}>
                <OrbitalSpinner size={24} />
                <OrbitalSpinner size={32} />
                <OrbitalSpinner size={40} />
                <OrbitalSpinner size={56} />
                <OrbitalSpinner size={80} />
            </Box>
        </Box>
        
        <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
            <Typography variant="h6">In Context</Typography>
            
            {/* Dark theme example */}
            <Box sx={{ p: 3, bgcolor: "#000", borderRadius: 2, display: "flex", alignItems: "center", gap: 2 }}>
                <OrbitalSpinner size={24} />
                <Typography sx={{ color: "#fff" }}>Loading swarm intelligence...</Typography>
            </Box>
            
            {/* Card example */}
            <Box sx={{ p: 3, bgcolor: "background.paper", borderRadius: 2, boxShadow: 2 }}>
                <Box sx={{ display: "flex", alignItems: "center", gap: 2, mb: 2 }}>
                    <OrbitalSpinner size={20} />
                    <Typography variant="body2">Orchestrating agents...</Typography>
                </Box>
                <Typography variant="caption" color="text.secondary">
                    The orbital motion represents your three-tier AI architecture with autonomous agents
                </Typography>
            </Box>
        </Box>
        
        <Typography variant="body2" color="text.secondary">
            The Orbital Spinner features:
            • 3 orbiting particles representing your 3-tier architecture
            • Cyan neon glow matching your landing page aesthetic
            • Different orbital speeds showing hierarchical coordination
            • Space-inspired design aligning with your brand
        </Typography>
    </Box>
);

// Performance comparison
export const PerformanceComparison = () => (
    <Box sx={{ p: 4, display: "flex", flexDirection: "column", gap: 3 }}>
        <Typography variant="h6">Multiple Spinners Performance Test</Typography>
        <Typography variant="body2" color="text.secondary">
            All spinners use CSS animations for optimal performance
        </Typography>
        
        <Box sx={{ display: "grid", gridTemplateColumns: "repeat(10, 1fr)", gap: 2 }}>
            {Array.from({ length: 50 }).map((_, i) => (
                <CircularProgress 
                    key={i} 
                    size={24} 
                    variant={i % 2 === 0 ? "primary" : "secondary"}
                    dualRing={i % 3 === 0}
                />
            ))}
        </Box>
    </Box>
);