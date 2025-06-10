import type { Meta } from "@storybook/react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Divider from "@mui/material/Divider";
import { useState } from "react";
import { IconCommon } from "../../icons/Icons.js";
import { HybridLoadableButton } from "./HybridLoadableButton.js";
import { TailwindButton } from "./TailwindButton.js";

const meta: Meta = {
    title: "Components/Buttons/Tailwind Migration Demo",
    parameters: {
        layout: "centered",
    },
};

export default meta;

export const ComprehensiveMigrationDemo = () => {
    const [loadingStates, setLoadingStates] = useState<Record<string, boolean>>({});

    const handleClick = (key: string) => {
        setLoadingStates(prev => ({ ...prev, [key]: true }));
        setTimeout(() => {
            setLoadingStates(prev => ({ ...prev, [key]: false }));
        }, 2000);
    };

    return (
        <Box sx={{ p: 4, maxWidth: 1200 }}>
            <Typography variant="h4" sx={{ mb: 4 }}>
                Tailwind CSS Button Migration Strategy
            </Typography>

            {/* Section 1: Direct Comparison */}
            <Box sx={{ mb: 6 }}>
                <Typography variant="h5" sx={{ mb: 3 }}>
                    1. Direct Comparison: MUI vs Tailwind
                </Typography>
                
                <Box sx={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 4 }}>
                    <Box>
                        <Typography variant="h6" sx={{ mb: 2 }}>MUI Button (Current)</Typography>
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                            <HybridLoadableButton
                                useTailwind={false}
                                variant="contained"
                                isLoading={loadingStates["mui-1"] || false}
                                onClick={() => handleClick("mui-1")}
                                startIcon={<IconCommon name="Save" />}
                            >
                                Save Changes
                            </HybridLoadableButton>
                            
                            <HybridLoadableButton
                                useTailwind={false}
                                variant="outlined"
                                isLoading={loadingStates["mui-2"] || false}
                                onClick={() => handleClick("mui-2")}
                            >
                                Cancel
                            </HybridLoadableButton>
                            
                            <HybridLoadableButton
                                useTailwind={false}
                                variant="text"
                                isLoading={loadingStates["mui-3"] || false}
                                onClick={() => handleClick("mui-3")}
                            >
                                Learn More
                            </HybridLoadableButton>
                        </Box>
                    </Box>
                    
                    <Box>
                        <Typography variant="h6" sx={{ mb: 2 }}>Tailwind Button (New)</Typography>
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                            <HybridLoadableButton
                                useTailwind={true}
                                variant="contained"
                                isLoading={loadingStates["tw-1"] || false}
                                onClick={() => handleClick("tw-1")}
                                startIcon={<IconCommon name="Save" />}
                            >
                                Save Changes
                            </HybridLoadableButton>
                            
                            <HybridLoadableButton
                                useTailwind={true}
                                variant="outlined"
                                isLoading={loadingStates["tw-2"] || false}
                                onClick={() => handleClick("tw-2")}
                            >
                                Cancel
                            </HybridLoadableButton>
                            
                            <HybridLoadableButton
                                useTailwind={true}
                                variant="text"
                                isLoading={loadingStates["tw-3"] || false}
                                onClick={() => handleClick("tw-3")}
                            >
                                Learn More
                            </HybridLoadableButton>
                        </Box>
                    </Box>
                </Box>
            </Box>

            <Divider sx={{ my: 4 }} />

            {/* Section 2: Professional Tailwind Component */}
            <Box sx={{ mb: 6 }}>
                <Typography variant="h5" sx={{ mb: 3 }}>
                    2. Professional Tailwind Button Component
                </Typography>
                
                <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                    Our custom TailwindButton component provides additional variants and better performance:
                </Typography>
                
                <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap", mb: 3 }}>
                    <TailwindButton
                        variant="primary"
                        isLoading={loadingStates["pro-1"] || false}
                        onClick={() => handleClick("pro-1")}
                        startIcon={<IconCommon name="Save" />}
                    >
                        Primary Action
                    </TailwindButton>
                    
                    <TailwindButton
                        variant="secondary"
                        isLoading={loadingStates["pro-2"] || false}
                        onClick={() => handleClick("pro-2")}
                    >
                        Secondary
                    </TailwindButton>
                    
                    <TailwindButton
                        variant="outline"
                        isLoading={loadingStates["pro-3"] || false}
                        onClick={() => handleClick("pro-3")}
                    >
                        Outline
                    </TailwindButton>
                    
                    <TailwindButton
                        variant="ghost"
                        isLoading={loadingStates["pro-4"] || false}
                        onClick={() => handleClick("pro-4")}
                    >
                        Ghost
                    </TailwindButton>
                    
                    <TailwindButton
                        variant="danger"
                        isLoading={loadingStates["pro-5"] || false}
                        onClick={() => handleClick("pro-5")}
                        startIcon={<IconCommon name="Delete" />}
                    >
                        Delete
                    </TailwindButton>
                </Box>
                
                <Typography variant="body2" color="text.secondary">
                    Sizes available: sm, md, lg, icon
                </Typography>
                <Box sx={{ display: "flex", gap: 2, alignItems: "center", mt: 2 }}>
                    <TailwindButton size="sm" variant="secondary">Small</TailwindButton>
                    <TailwindButton size="md" variant="secondary">Medium</TailwindButton>
                    <TailwindButton size="lg" variant="secondary">Large</TailwindButton>
                    <TailwindButton size="icon" variant="secondary"><IconCommon name="Save" /></TailwindButton>
                </Box>
            </Box>

            <Divider sx={{ my: 4 }} />

            {/* Section 3: Migration Benefits */}
            <Box sx={{ mb: 6 }}>
                <Typography variant="h5" sx={{ mb: 3 }}>
                    3. Migration Benefits
                </Typography>
                
                <Box sx={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(250px, 1fr))", gap: 3 }}>
                    <Box sx={{ p: 3, bgcolor: "background.paper", borderRadius: 2 }}>
                        <Typography variant="h6" sx={{ mb: 1 }}>🚀 Performance</Typography>
                        <Typography variant="body2" color="text.secondary">
                            No runtime CSS-in-JS calculations. Styles are compiled at build time.
                        </Typography>
                    </Box>
                    
                    <Box sx={{ p: 3, bgcolor: "background.paper", borderRadius: 2 }}>
                        <Typography variant="h6" sx={{ mb: 1 }}>📦 Bundle Size</Typography>
                        <Typography variant="body2" color="text.secondary">
                            Smaller bundle with tree-shaken utilities instead of entire component library.
                        </Typography>
                    </Box>
                    
                    <Box sx={{ p: 3, bgcolor: "background.paper", borderRadius: 2 }}>
                        <Typography variant="h6" sx={{ mb: 1 }}>🎨 Customization</Typography>
                        <Typography variant="body2" color="text.secondary">
                            Easy to extend with additional variants and responsive modifiers.
                        </Typography>
                    </Box>
                    
                    <Box sx={{ p: 3, bgcolor: "background.paper", borderRadius: 2 }}>
                        <Typography variant="h6" sx={{ mb: 1 }}>♿ Accessibility</Typography>
                        <Typography variant="body2" color="text.secondary">
                            Built-in ARIA attributes, keyboard navigation, and focus management.
                        </Typography>
                    </Box>
                </Box>
            </Box>

            <Divider sx={{ my: 4 }} />

            {/* Section 4: Migration Code Example */}
            <Box>
                <Typography variant="h5" sx={{ mb: 3 }}>
                    4. Migration Code Example
                </Typography>
                
                <Box sx={{ p: 3, bgcolor: "grey.100", borderRadius: 2, fontFamily: "monospace" }}>
                    <Typography variant="body2" sx={{ whiteSpace: "pre", color: "text.primary" }}>
{`// Before (MUI)
<Button 
  variant="contained" 
  startIcon={<IconCommon name="Save" />}
  onClick={handleSave}
>
  Save
</Button>

// During Migration (Hybrid)
<HybridLoadableButton 
  useTailwind={true}  // Toggle this flag
  variant="contained"
  startIcon={<IconCommon name="Save" />}
  onClick={handleSave}
>
  Save
</HybridLoadableButton>

// After Migration (Pure Tailwind)
<TailwindButton 
  variant="primary"
  startIcon={<IconCommon name="Save" />}
  onClick={handleSave}
>
  Save
</TailwindButton>`}
                    </Typography>
                </Box>
            </Box>
        </Box>
    );
};