import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import { IconButton } from "../components/buttons/IconButton.js";
import InputAdornment from "@mui/material/InputAdornment";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import { Tooltip } from "../components/Tooltip/Tooltip.js";
import type { Meta } from "@storybook/react";
import React, { useEffect, useRef, useState } from "react";
import { IconCommon, IconFavicon, type IconInfo, IconRoutine, IconService, IconText } from "./Icons.js";
import { Slider } from "../components/inputs/Slider.js";
import { iconNames as commonIconNames } from "./types/commonIcons.js";
import { iconNames as routineIconNames } from "./types/routineIcons.js";
import { iconNames as serviceIconNames } from "./types/serviceIcons.js";
import { iconNames as textIconNames } from "./types/textIcons.js";
import { paddedDecorator } from "../__test/helpers/storybookDecorators.tsx";

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
    decorators: [paddedDecorator(2)],
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
                        decorative="false"
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
                            decorative="false"
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
                        decorative="true"
                    />
                    <AttributeDisplay componentRef={decorativeRef} />
                </Box>
                <Box>
                    <h4>Meaningful (announced by screen readers)</h4>
                    <IconComponent
                        ref={meaningfulRef}
                        name={name}
                        decorative="false"
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

// Add new style constants for the AllIcons story
const iconGridStyle = {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fill, minmax(120px, 1fr))",
    gap: 2,
} as const;

const iconNameStyle = {
    fontSize: "0.75rem",
    color: "text.secondary",
    textAlign: "center",
} as const;

const sliderContainerStyle = {
    display: "flex",
    alignItems: "center",
    gap: 2,
    mb: 4,
} as const;

// Add new constants for the AllIcons story
const DEFAULT_ICON_SIZE = 24;
const MIN_ICON_SIZE = 16;
const MAX_ICON_SIZE = 64;
const ICON_SIZE_STEP = 4;

const ICON_SIZE_MARKS = [
    { value: 16, label: "16px" },
    { value: 24, label: "24px" },
    { value: 32, label: "32px" },
    { value: 48, label: "48px" },
    { value: 64, label: "64px" },
] as const;

const controlPanelStyle = {
    display: "flex",
    flexDirection: "column",
    gap: 2,
    mb: 4,
} as const;

const controlRowStyle = {
    display: "flex",
    alignItems: "center",
    gap: 2,
} as const;

const flexGrowStyle = {
    flexGrow: 1,
    maxWidth: 300,
} as const;

const categoryVisibilityStyle = {
    display: "block",
} as const;

const categoryHiddenStyle = {
    display: "none",
} as const;

const searchInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="Search" size={20} />
        </InputAdornment>
    ),
} as const;

// Function to filter icons by search term
function filterIconsByName(names: readonly string[], searchTerm: string): string[] {
    const normalizedSearch = searchTerm.toLowerCase();
    return names.filter(name => name.toLowerCase().includes(normalizedSearch));
}

// Update AllIcons story with search and proper constants
export function AllIcons() {
    const [iconSize, setIconSize] = useState(DEFAULT_ICON_SIZE);
    const [searchTerm, setSearchTerm] = useState("");

    // Memoize filtered icon names
    const filteredCommonIcons = React.useMemo(
        () => filterIconsByName(commonIconNames, searchTerm),
        [searchTerm],
    );
    const filteredRoutineIcons = React.useMemo(
        () => filterIconsByName(routineIconNames, searchTerm),
        [searchTerm],
    );
    const filteredServiceIcons = React.useMemo(
        () => filterIconsByName(serviceIconNames, searchTerm),
        [searchTerm],
    );
    const filteredTextIcons = React.useMemo(
        () => filterIconsByName(textIconNames, searchTerm),
        [searchTerm],
    );

    // Handler functions
    const handleSizeChange = React.useCallback((newValue: number) => {
        setIconSize(newValue);
    }, []);

    const handleSearchChange = React.useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setSearchTerm(event.target.value);
    }, []);

    // Update renderIconCategory function
    const renderIconCategory = React.useCallback((
        title: string,
        IconComponent: typeof IconCommon | typeof IconRoutine | typeof IconService | typeof IconText,
        icons: string[],
    ) => (
        <Box sx={icons.length ? categoryVisibilityStyle : categoryHiddenStyle}>
            <h2>{title} ({icons.length})</h2>
            <Box sx={iconGridStyle}>
                {icons.map((name) => (
                    <Stack key={name} alignItems="center" spacing={1}>
                        <Tooltip title={name}>
                            <IconComponent name={name} size={iconSize} />
                        </Tooltip>
                        <Box sx={iconNameStyle}>
                            {name}
                        </Box>
                    </Stack>
                ))}
            </Box>
        </Box>
    ), [iconSize]);

    return (
        <Stack spacing={4}>
            <Box sx={controlPanelStyle}>
                <Box sx={controlRowStyle}>
                    <Box sx={flexGrowStyle}>
                        <Slider
                            value={iconSize}
                            onChange={handleSizeChange}
                            min={MIN_ICON_SIZE}
                            max={MAX_ICON_SIZE}
                            step={ICON_SIZE_STEP}
                            marks={ICON_SIZE_MARKS}
                            label={`Icon Size: ${iconSize}px`}
                            showValue={false}
                        />
                    </Box>
                </Box>
                <Box sx={controlRowStyle}>
                    <TextField
                        fullWidth
                        placeholder="Search icons..."
                        value={searchTerm}
                        onChange={handleSearchChange}
                        InputProps={searchInputProps}
                    />
                </Box>
            </Box>

            {renderIconCategory("Common Icons", IconCommon, filteredCommonIcons)}
            {renderIconCategory("Routine Icons", IconRoutine, filteredRoutineIcons)}
            {renderIconCategory("Service Icons", IconService, filteredServiceIcons)}
            {renderIconCategory("Text Icons", IconText, filteredTextIcons)}
        </Stack>
    );
}
AllIcons.parameters = {
    docs: {
        description: {
            story: "A comprehensive display of all available icons across all categories: Common, Routine, Service, and Text icons.",
        },
    },
};

// Add IconFavicon story
function Favicons() {
    const validUrls = [
        "https://vrooli.com",
        "https://www.google.com",
        "https://github.com",
        "https://facebook.com",
        "https://twitter.com",
        "https://linkedin.com",
    ];

    const invalidUrls = [
        "not-a-url",
        "mailto:user@example.com",
        "tel:+1234567890",
        "",
    ];

    const customFallbacks: IconInfo[] = [
        { name: "Website", type: "Common" },
        { name: "Bot", type: "Common" },
        { name: "User", type: "Common" },
        { name: "Team", type: "Common" },
    ];

    return (
        <Stack spacing={4}>
            <Box>
                <h3>Valid Website Favicons</h3>
                <Stack direction="row" spacing={4} alignItems="center">
                    {validUrls.map((url) => (
                        <Stack key={url} alignItems="center">
                            <IconFavicon href={url} size={24} />
                            <Box sx={labelStyle}>{new URL(url).hostname}</Box>
                        </Stack>
                    ))}
                </Stack>
            </Box>

            <Box>
                <h3>Invalid URLs (Default Fallback Icon)</h3>
                <Stack direction="row" spacing={4} alignItems="center">
                    {invalidUrls.map((url) => (
                        <Stack key={url} alignItems="center">
                            <IconFavicon href={url} size={24} />
                            <Box sx={labelStyle}>{url || "(empty string)"}</Box>
                        </Stack>
                    ))}
                </Stack>
            </Box>

            <Box>
                <h3>Custom Fallback Icons</h3>
                <Stack direction="row" spacing={4} alignItems="center">
                    {customFallbacks.map((fallbackIcon) => (
                        <Stack key={`${fallbackIcon.type}-${fallbackIcon.name}`} alignItems="center">
                            <IconFavicon
                                href="not-a-url"
                                size={24}
                                fallbackIcon={fallbackIcon}
                            />
                            <Box sx={labelStyle}>{fallbackIcon.name}</Box>
                        </Stack>
                    ))}
                </Stack>
            </Box>

            <Box>
                <h3>Custom Styling</h3>
                <Stack direction="row" spacing={4} alignItems="center">
                    <Stack alignItems="center">
                        <IconFavicon
                            href="https://github.com"
                            size={32}
                            fill="primary"
                        />
                        <Box sx={labelStyle}>Primary Color</Box>
                    </Stack>
                    <Stack alignItems="center">
                        <IconFavicon
                            href="https://github.com"
                            size={32}
                            fill="#FF5733"
                        />
                        <Box sx={labelStyle}>Custom Color</Box>
                    </Stack>
                    <Stack alignItems="center">
                        <IconFavicon
                            href="https://github.com"
                            size={48}
                            style={shadowAndScaleStyle}
                        />
                        <Box sx={labelStyle}>With Shadow & Scale</Box>
                    </Stack>
                    <Stack alignItems="center">
                        <IconFavicon
                            href="https://github.com"
                            size={32}
                            style={spinAnimationStyle}
                        />
                        <Box sx={labelStyle}>Animated</Box>
                        <style>{spinKeyframes}</style>
                    </Stack>
                </Stack>
            </Box>

            <Box>
                <h3>Accessibility Examples</h3>
                <Stack direction="row" spacing={4} alignItems="center">
                    <Stack alignItems="center">
                        <IconFavicon
                            href="https://github.com"
                            size={24}
                            decorative="true"
                        />
                        <Box sx={labelStyle}>Decorative (hidden from screen readers)</Box>
                    </Stack>
                    <Stack alignItems="center">
                        <IconFavicon
                            href="https://github.com"
                            size={24}
                            decorative="false"
                            aria-label="GitHub favicon"
                        />
                        <Box sx={labelStyle}>With aria-label</Box>
                    </Stack>
                </Stack>
            </Box>

            <Box>
                <h3>In Components</h3>
                <Stack direction="row" spacing={4} alignItems="center">
                    <Button
                        variant="contained"
                        startIcon={
                            <IconFavicon href="https://github.com" size={16} />
                        }
                    >
                        Open in GitHub
                    </Button>
                    <TextField
                        label="Website URL"
                        defaultValue="https://github.com"
                        InputProps={{
                            startAdornment: (
                                <InputAdornment position="start">
                                    <IconFavicon href="https://github.com" size={16} />
                                </InputAdornment>
                            ),
                        }}
                        sx={searchFieldContainerStyle}
                    />
                </Stack>
            </Box>
        </Stack>
    );
}

Favicons.parameters = {
    docs: {
        description: {
            story: "IconFavicon component for displaying website favicons with various configurations and use cases.",
        },
    },
}; 
