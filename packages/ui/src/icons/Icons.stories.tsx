import { Box, Button, ButtonGroup, IconButton, InputAdornment, Stack, TextField, Tooltip } from "@mui/material";
import type { Meta } from "@storybook/react";
import React, { useEffect, useRef, useState } from "react";
import { IconCommon, IconRoutine, IconService, IconText } from "./Icons.js";

// Styles as constants to avoid linter errors

const cursorPointerStyle = {
    cursor: "pointer",
} as const;

const searchFieldContainerStyle = {
    maxWidth: 300,
} as const;

const githubButtonStyle = {
    bgcolor: "#171515",
    "&:hover": { bgcolor: "#2b2b2b" },
} as const;

const googleButtonStyle = {
    bgcolor: "#4285f4",
    "&:hover": { bgcolor: "#357abd" },
} as const;

// Add these style constants after the other style constants
const attributeDisplayStyle = {
    mt: 1,
    p: 1.5,
    bgcolor: "grey.100",
    borderRadius: 1,
    fontFamily: "monospace",
    fontSize: "0.75rem",
} as const;

const attributeNameStyle = {
    color: "primary.main",
} as const;

const attributeValueStyle = {
    color: "success.main",
} as const;

const attributeLineStyle = {
    color: "text.secondary",
} as const;

const labelStyle = {
    mt: 1,
    fontSize: "0.75rem",
    color: "text.secondary",
} as const;

// Add these style constants after the other style constants
const shadowAndScaleStyle = {
    filter: "drop-shadow(2px 2px 2px rgba(0,0,0,0.2))",
    transform: "scale(1.2)",
} as const;

const spinAnimationStyle = {
    animation: "spin 2s linear infinite",
} as const;

const spinKeyframes = `
@keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
}
`;

// Input props as functions to avoid linter errors
function createSearchInputProps() {
    return {
        startAdornment: (
            <InputAdornment position="start">
                <IconCommon name="Search" size={20} />
            </InputAdornment>
        ),
    };
}

function createPasswordInputProps() {
    return {
        endAdornment: (
            <InputAdornment position="end">
                <IconButton edge="end" size="small">
                    <IconCommon name="Visible" />
                </IconButton>
            </InputAdornment>
        ),
    };
}

const meta = {
    title: "Components/Icons",
    component: IconCommon,
    parameters: {
        docs: {
            description: {
                component: "Icon components for displaying SVG sprites with various configurations.",
            },
        },
    },
    decorators: [
        (Story) => (
            <Box p={2} overflow="auto">
                <Story />
            </Box>
        ),
    ],
} satisfies Meta<typeof IconCommon>;

export default meta;

// Update the AttributeDisplay component
function AttributeDisplay({ componentRef }: { componentRef: React.RefObject<SVGSVGElement> }) {
    const [attrs, setAttrs] = useState<Record<string, string>>({});

    useEffect(() => {
        console.log("componentRef", componentRef.current);
        if (componentRef.current) {
            const element = componentRef.current;
            const attributes: Record<string, string> = {};
            Array.from(element.attributes).forEach(attr => {
                attributes[attr.name] = attr.value;
            });
            setAttrs(attributes);
        }
    }, [componentRef]);

    return (
        <Box sx={attributeDisplayStyle}>
            {Object.entries(attrs).map(([key, value]) => (
                <Box key={key} sx={attributeLineStyle}>
                    <Box component="span" sx={attributeNameStyle}>{key}</Box>
                    =&quot;
                    <Box component="span" sx={attributeValueStyle}>{value}</Box>
                    &quot;
                </Box>
            ))}
        </Box>
    );
}

// Constants for icon sizes
// eslint-disable-next-line no-magic-numbers
const ICON_SIZES = [16, 24, 32, 48] as const;

// Constants for icon colors
const ICON_COLORS = [
    { fill: "primary", label: "primary" },
    { fill: "secondary", label: "secondary" },
    { fill: "error", label: "error" },
    { fill: "text.secondary", label: "text.secondary" },
    { fill: "grey.500", label: "grey.500" },
    { fill: "#FF5733", label: "#FF5733" },
] as const;

// Reusable click handler
function handleIconClick() {
    alert("Icon clicked!");
}

// Add after imports
type IconProps = {
    name: string;
    size?: number;
    fill?: string;
    style?: React.CSSProperties;
    decorative?: boolean;
    "aria-label"?: string;
    onClick?: () => void;
    ref?: React.Ref<SVGSVGElement>;
};

type IconComponent = React.ComponentType<IconProps>;

type IconSectionProps = {
    IconComponent: IconComponent;
    name: string;
};

// Update component type definitions
function IconSizeSection({ IconComponent, name }: IconSectionProps) {
    return (
        <Box>
            <h3>Sizes</h3>
            <Stack direction="row" spacing={4} alignItems="center">
                {ICON_SIZES.map((size) => (
                    <Stack key={size} alignItems="center">
                        <IconComponent name={name} size={size} />
                        <Box sx={labelStyle}>{size}px</Box>
                    </Stack>
                ))}
            </Stack>
        </Box>
    );
}

function IconColorSection({ IconComponent, name }: IconSectionProps) {
    return (
        <Box>
            <h3>Colors</h3>
            <Stack direction="row" spacing={4} alignItems="center">
                {ICON_COLORS.map(({ fill, label }) => (
                    <Stack key={label} alignItems="center">
                        <IconComponent name={name} fill={fill} />
                        <Box sx={labelStyle}>{label}</Box>
                    </Stack>
                ))}
            </Stack>
        </Box>
    );
}

function IconInteractiveSection({ IconComponent, name }: IconSectionProps) {
    return (
        <Box>
            <h3>Interactive Example</h3>
            <Stack direction="row" spacing={4} alignItems="center">
                <Stack alignItems="center">
                    <IconComponent
                        name={name}
                        style={cursorPointerStyle}
                        decorative={false}
                        aria-label={`${name} action`}
                        onClick={handleIconClick}
                    />
                    <Box sx={labelStyle}>Clickable</Box>
                </Stack>
                <Stack alignItems="center">
                    <Tooltip title="Click me!">
                        <IconComponent
                            name={name}
                            style={cursorPointerStyle}
                            decorative={false}
                            aria-label={`${name} with tooltip`}
                            onClick={handleIconClick}
                        />
                    </Tooltip>
                    <Box sx={labelStyle}>With Tooltip</Box>
                </Stack>
            </Stack>
        </Box>
    );
}

function IconCustomStyleSection({ IconComponent, name }: IconSectionProps) {
    return (
        <Box>
            <h3>Custom Styling</h3>
            <Stack direction="row" spacing={4} alignItems="center">
                <Stack alignItems="center">
                    <IconComponent
                        name={name}
                        style={shadowAndScaleStyle}
                    />
                    <Box sx={labelStyle}>With Shadow & Scale</Box>
                </Stack>
                <Stack alignItems="center">
                    <IconComponent
                        name={name}
                        style={spinAnimationStyle}
                    />
                    <Box sx={labelStyle}>Animated</Box>
                    <style>{spinKeyframes}</style>
                </Stack>
            </Stack>
        </Box>
    );
}

function IconAccessibilitySection({ IconComponent, name }: IconSectionProps) {
    const decorativeRef = useRef<SVGSVGElement>(null);
    const meaningfulRef = useRef<SVGSVGElement>(null);

    return (
        <Box>
            <h3>Accessibility</h3>
            <Stack spacing={4}>
                <Box>
                    <h4>Decorative (hidden from screen readers)</h4>
                    <IconComponent
                        ref={decorativeRef}
                        name={name}
                        decorative={true}
                    />
                    <AttributeDisplay componentRef={decorativeRef} />
                </Box>
                <Box>
                    <h4>Meaningful (announced by screen readers)</h4>
                    <IconComponent
                        ref={meaningfulRef}
                        name={name}
                        decorative={false}
                        aria-label={`${name} button`}
                    />
                    <AttributeDisplay componentRef={meaningfulRef} />
                </Box>
            </Stack>
        </Box>
    );
}

// Update CommonIcons to use the new component
export function CommonIcons() {
    return (
        <Stack spacing={2}>
            <IconSizeSection IconComponent={IconCommon as IconComponent} name="Search" />
            <IconColorSection IconComponent={IconCommon as IconComponent} name="Search" />
            <IconInteractiveSection IconComponent={IconCommon as IconComponent} name="Search" />
            <IconCustomStyleSection IconComponent={IconCommon as IconComponent} name="Search" />
            <IconAccessibilitySection IconComponent={IconCommon as IconComponent} name="Search" />
        </Stack>
    );
}

CommonIcons.parameters = {
    docs: {
        description: {
            story: "Common icons with various sizes, colors, and accessibility settings.",
        },
    },
};

// Update RoutineIcons to use the new component
export function RoutineIcons() {
    return (
        <Stack spacing={2}>
            <IconSizeSection IconComponent={IconRoutine as IconComponent} name="UnlinkedNodes" />
            <IconColorSection IconComponent={IconRoutine as IconComponent} name="UnlinkedNodes" />
            <IconInteractiveSection IconComponent={IconRoutine as IconComponent} name="UnlinkedNodes" />
            <IconCustomStyleSection IconComponent={IconRoutine as IconComponent} name="UnlinkedNodes" />
            <IconAccessibilitySection IconComponent={IconRoutine as IconComponent} name="UnlinkedNodes" />
        </Stack>
    );
}

// Update ServiceIcons to use the new component
export function ServiceIcons() {
    return (
        <Stack spacing={2}>
            <IconSizeSection IconComponent={IconService as IconComponent} name="GitHub" />
            <IconColorSection IconComponent={IconService as IconComponent} name="GitHub" />
            <IconInteractiveSection IconComponent={IconService as IconComponent} name="GitHub" />
            <IconCustomStyleSection IconComponent={IconService as IconComponent} name="GitHub" />
            <IconAccessibilitySection IconComponent={IconService as IconComponent} name="GitHub" />
        </Stack>
    );
}

// Update TextIcons to use the new component
export function TextIcons() {
    return (
        <Stack spacing={2}>
            <IconSizeSection IconComponent={IconText as IconComponent} name="Bold" />
            <IconColorSection IconComponent={IconText as IconComponent} name="Bold" />
            <IconInteractiveSection IconComponent={IconText as IconComponent} name="Bold" />
            <IconCustomStyleSection IconComponent={IconText as IconComponent} name="Bold" />
            <IconAccessibilitySection IconComponent={IconText as IconComponent} name="Bold" />
        </Stack>
    );
}

// Add new story after the existing stories
export function IconsInComponents() {
    return (
        <Stack spacing={4}>
            <Box>
                <h3>Icons in Buttons</h3>
                <Stack direction="row" spacing={2} alignItems="center">
                    <Button
                        variant="contained"
                        startIcon={<IconCommon name="Search" />}
                    >
                        Search
                    </Button>
                    <Button
                        variant="outlined"
                        endIcon={<IconCommon name="ArrowRight" />}
                    >
                        Next
                    </Button>
                    <ButtonGroup variant="contained">
                        <Button startIcon={<IconText name="Bold" />}>
                            Bold
                        </Button>
                        <Button startIcon={<IconText name="Italic" />}>
                            Italic
                        </Button>
                    </ButtonGroup>
                </Stack>
            </Box>

            <Box>
                <h3>Icon Buttons</h3>
                <Stack direction="row" spacing={2} alignItems="center">
                    <Tooltip title="Search">
                        <IconButton>
                            <IconCommon name="Search" />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Settings">
                        <IconButton color="primary">
                            <IconCommon name="Settings" />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Delete">
                        <IconButton color="error" size="small">
                            <IconCommon name="Delete" />
                        </IconButton>
                    </Tooltip>
                </Stack>
            </Box>

            <Box>
                <h3>Input Adornments</h3>
                <Stack spacing={2} sx={searchFieldContainerStyle}>
                    <TextField
                        label="Search"
                        InputProps={createSearchInputProps()}
                    />
                    <TextField
                        label="Password"
                        type="password"
                        InputProps={createPasswordInputProps()}
                    />
                </Stack>
            </Box>

            <Box>
                <h3>Service Integration Examples</h3>
                <Stack direction="row" spacing={2} alignItems="center">
                    <Button
                        variant="contained"
                        startIcon={
                            <Box sx={serviceIconContainerStyle}>
                                <IconService name="GitHub" size={16} />
                            </Box>
                        }
                        sx={githubButtonStyle}
                    >
                        Login with GitHub
                    </Button>
                    <Button
                        variant="contained"
                        startIcon={
                            <Box sx={serviceIconContainerStyle}>
                                <IconService name="Google" size={16} />
                            </Box>
                        }
                        sx={googleButtonStyle}
                    >
                        Login with Google
                    </Button>
                </Stack>
            </Box>

            <Box>
                <h3>Routine Actions</h3>
                <Stack direction="row" spacing={2} alignItems="center">
                    <Tooltip title="Link Nodes">
                        <IconButton color="primary" size="large">
                            <IconRoutine name="UnlinkedNodes" size={32} />
                        </IconButton>
                    </Tooltip>
                    <Button
                        variant="outlined"
                        startIcon={<IconRoutine name="MoveNode" />}
                    >
                        Move Node
                    </Button>
                </Stack>
            </Box>
        </Stack>
    );
}

IconsInComponents.parameters = {
    docs: {
        description: {
            story: "Examples of icons used within various Material-UI components and common UI patterns.",
        },
    },
};

// Add new style constant for icon container
const serviceIconContainerStyle = {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: "#ffffff",
    borderRadius: "50%",
    width: 24,
    height: 24,
    padding: "4px",
} as const; 
