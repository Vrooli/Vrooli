import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import RadioGroup from "@mui/material/RadioGroup";
import FormControlLabel from "@mui/material/FormControlLabel";
import Radio from "@mui/material/Radio";
import TextField from "@mui/material/TextField";
import { Button } from "../buttons/Button.js";
import { IconButton } from "../buttons/IconButton.js";
import { IconCommon } from "../../icons/Icons.js";
import { Tooltip } from "./Tooltip.js";
import type { TooltipPlacement } from "./Tooltip.js";
import { Switch } from "../inputs/Switch/Switch.js";

const meta: Meta<typeof Tooltip> = {
    title: "Components/Dialogs/Tooltip",
    component: Tooltip,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Interactive Tooltip Playground
export const TooltipShowcase: Story = {
    render: () => {
        const [placement, setPlacement] = useState<TooltipPlacement>("top");
        const [arrow, setArrow] = useState(true);
        const [enterDelay, setEnterDelay] = useState(100);
        const [leaveDelay, setLeaveDelay] = useState(0);
        const [controlledOpen, setControlledOpen] = useState<boolean | undefined>(undefined);
        const [tooltipText, setTooltipText] = useState("This is a helpful tooltip!");

        const placements: TooltipPlacement[] = [
            "top", "top-start", "top-end",
            "bottom", "bottom-start", "bottom-end",
            "left", "left-start", "left-end",
            "right", "right-start", "right-end",
        ];

        return (
            <Box sx={{ 
                p: 2, 
                height: "100vh", 
                overflow: "auto",
                bgcolor: "background.default", 
            }}>
                <Box sx={{ 
                    display: "flex", 
                    gap: 2, 
                    flexDirection: "column",
                    maxWidth: 1200, 
                    mx: "auto", 
                }}>
                    {/* Controls Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Tooltip Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)", lg: "repeat(3, 1fr)" },
                            gap: 3, 
                        }}>
                            {/* Placement Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Placement</FormLabel>
                                <RadioGroup
                                    value={placement}
                                    onChange={(e) => setPlacement(e.target.value as TooltipPlacement)}
                                    sx={{ gap: 0.5, maxHeight: 200, overflow: "auto" }}
                                >
                                    {placements.map(p => (
                                        <FormControlLabel 
                                            key={p} 
                                            value={p} 
                                            control={<Radio size="small" />} 
                                            label={p} 
                                            sx={{ m: 0 }} 
                                        />
                                    ))}
                                </RadioGroup>
                            </FormControl>

                            {/* Options */}
                            <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                                <FormControl component="fieldset" size="small">
                                    <Switch
                                        checked={arrow}
                                        onChange={setArrow}
                                        size="sm"
                                        label="Show Arrow"
                                        labelPosition="right"
                                    />
                                </FormControl>

                                <TextField
                                    label="Enter Delay (ms)"
                                    type="number"
                                    value={enterDelay}
                                    onChange={(e) => setEnterDelay(Number(e.target.value))}
                                    size="small"
                                    inputProps={{ min: 0, max: 2000 }}
                                />

                                <TextField
                                    label="Leave Delay (ms)"
                                    type="number"
                                    value={leaveDelay}
                                    onChange={(e) => setLeaveDelay(Number(e.target.value))}
                                    size="small"
                                    inputProps={{ min: 0, max: 2000 }}
                                />
                            </Box>

                            {/* Controlled State */}
                            <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                                <FormControl component="fieldset" size="small">
                                    <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Controlled State</FormLabel>
                                    <RadioGroup
                                        value={controlledOpen === undefined ? "uncontrolled" : controlledOpen ? "open" : "closed"}
                                        onChange={(e) => {
                                            const value = e.target.value;
                                            setControlledOpen(
                                                value === "uncontrolled" ? undefined :
                                                value === "open" ? true : false,
                                            );
                                        }}
                                        sx={{ gap: 0.5 }}
                                    >
                                        <FormControlLabel value="uncontrolled" control={<Radio size="small" />} label="Uncontrolled" sx={{ m: 0 }} />
                                        <FormControlLabel value="open" control={<Radio size="small" />} label="Always Open" sx={{ m: 0 }} />
                                        <FormControlLabel value="closed" control={<Radio size="small" />} label="Always Closed" sx={{ m: 0 }} />
                                    </RadioGroup>
                                </FormControl>

                                <TextField
                                    label="Tooltip Text"
                                    value={tooltipText}
                                    onChange={(e) => setTooltipText(e.target.value)}
                                    size="small"
                                    multiline
                                    rows={2}
                                />
                            </Box>
                        </Box>
                    </Box>

                    {/* Demo Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        minHeight: 400,
                        display: "flex",
                        flexDirection: "column",
                        alignItems: "center",
                        justifyContent: "center",
                        gap: 2,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Tooltip Demo</Typography>
                        
                        <Box sx={{ 
                            display: "flex", 
                            alignItems: "center", 
                            justifyContent: "center",
                            minHeight: 200,
                            minWidth: 300,
                        }}>
                            <Tooltip
                                title={tooltipText}
                                placement={placement}
                                arrow={arrow}
                                enterDelay={enterDelay}
                                leaveDelay={leaveDelay}
                                open={controlledOpen}
                            >
                                <Button variant="primary" size="lg">
                                    Hover over me!
                                </Button>
                            </Tooltip>
                        </Box>
                    </Box>

                    {/* Examples Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>Common Examples</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)" },
                            gap: 4,
                            alignItems: "center",
                            justifyItems: "center",
                        }}>
                            {/* Button with Tooltip */}
                            <Box sx={{ textAlign: "center" }}>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>Button</Typography>
                                <Tooltip title="Save your changes" placement="top">
                                    <Button variant="primary" startIcon={<IconCommon name="Save" />}>
                                        Save
                                    </Button>
                                </Tooltip>
                            </Box>

                            {/* Icon Button with Tooltip */}
                            <Box sx={{ textAlign: "center" }}>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>Icon Button</Typography>
                                <Tooltip title="Delete item" placement="top">
                                    <IconButton variant="outlined" color="error">
                                        <IconCommon name="Delete" />
                                    </IconButton>
                                </Tooltip>
                            </Box>

                            {/* Text with Tooltip */}
                            <Box sx={{ textAlign: "center" }}>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>Text</Typography>
                                <Tooltip title="This text has additional information" placement="top">
                                    <Typography 
                                        component="span"
                                        sx={{ 
                                            textDecoration: "underline",
                                            textDecorationStyle: "dotted",
                                            cursor: "help",
                                        }}
                                    >
                                        Hover for info
                                    </Typography>
                                </Tooltip>
                            </Box>

                            {/* Disabled Element */}
                            <Box sx={{ textAlign: "center" }}>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>Disabled</Typography>
                                <Tooltip title="This action is currently unavailable" placement="top">
                                    <span>
                                        <Button variant="secondary" disabled>
                                            Disabled
                                        </Button>
                                    </span>
                                </Tooltip>
                            </Box>

                            {/* Long Content */}
                            <Box sx={{ textAlign: "center" }}>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>Long Content</Typography>
                                <Tooltip 
                                    title="This is a longer tooltip with more detailed information that wraps to multiple lines when needed."
                                    placement="top"
                                >
                                    <Button variant="outline">
                                        Long tooltip
                                    </Button>
                                </Tooltip>
                            </Box>

                            {/* Custom Delay */}
                            <Box sx={{ textAlign: "center" }}>
                                <Typography variant="subtitle2" sx={{ mb: 2 }}>Custom Delay</Typography>
                                <Tooltip 
                                    title="This appears after 1 second"
                                    placement="top"
                                    enterDelay={1000}
                                    leaveDelay={500}
                                >
                                    <Button variant="ghost">
                                        Delayed tooltip
                                    </Button>
                                </Tooltip>
                            </Box>
                        </Box>
                    </Box>

                    {/* Placement Grid */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>All Placements</Typography>
                        
                        <Box sx={{ 
                            display: "grid",
                            gridTemplateColumns: "repeat(5, 1fr)",
                            gridTemplateRows: "repeat(5, 1fr)",
                            gap: 2,
                            minHeight: 400,
                            position: "relative",
                        }}>
                            {/* Top row */}
                            <Box sx={{ gridColumn: "2", gridRow: "1", display: "flex", justifyContent: "center" }}>
                                <Tooltip title="top-start" placement="top-start">
                                    <Button variant="outline" size="sm">TS</Button>
                                </Tooltip>
                            </Box>
                            <Box sx={{ gridColumn: "3", gridRow: "1", display: "flex", justifyContent: "center" }}>
                                <Tooltip title="top" placement="top">
                                    <Button variant="outline" size="sm">T</Button>
                                </Tooltip>
                            </Box>
                            <Box sx={{ gridColumn: "4", gridRow: "1", display: "flex", justifyContent: "center" }}>
                                <Tooltip title="top-end" placement="top-end">
                                    <Button variant="outline" size="sm">TE</Button>
                                </Tooltip>
                            </Box>

                            {/* Left column */}
                            <Box sx={{ gridColumn: "1", gridRow: "2", display: "flex", alignItems: "center", justifyContent: "center" }}>
                                <Tooltip title="left-start" placement="left-start">
                                    <Button variant="outline" size="sm">LS</Button>
                                </Tooltip>
                            </Box>
                            <Box sx={{ gridColumn: "1", gridRow: "3", display: "flex", alignItems: "center", justifyContent: "center" }}>
                                <Tooltip title="left" placement="left">
                                    <Button variant="outline" size="sm">L</Button>
                                </Tooltip>
                            </Box>
                            <Box sx={{ gridColumn: "1", gridRow: "4", display: "flex", alignItems: "center", justifyContent: "center" }}>
                                <Tooltip title="left-end" placement="left-end">
                                    <Button variant="outline" size="sm">LE</Button>
                                </Tooltip>
                            </Box>

                            {/* Center */}
                            <Box sx={{ gridColumn: "3", gridRow: "3", display: "flex", alignItems: "center", justifyContent: "center" }}>
                                <Typography variant="h6" sx={{ color: "text.secondary" }}>
                                    Target
                                </Typography>
                            </Box>

                            {/* Right column */}
                            <Box sx={{ gridColumn: "5", gridRow: "2", display: "flex", alignItems: "center", justifyContent: "center" }}>
                                <Tooltip title="right-start" placement="right-start">
                                    <Button variant="outline" size="sm">RS</Button>
                                </Tooltip>
                            </Box>
                            <Box sx={{ gridColumn: "5", gridRow: "3", display: "flex", alignItems: "center", justifyContent: "center" }}>
                                <Tooltip title="right" placement="right">
                                    <Button variant="outline" size="sm">R</Button>
                                </Tooltip>
                            </Box>
                            <Box sx={{ gridColumn: "5", gridRow: "4", display: "flex", alignItems: "center", justifyContent: "center" }}>
                                <Tooltip title="right-end" placement="right-end">
                                    <Button variant="outline" size="sm">RE</Button>
                                </Tooltip>
                            </Box>

                            {/* Bottom row */}
                            <Box sx={{ gridColumn: "2", gridRow: "5", display: "flex", justifyContent: "center" }}>
                                <Tooltip title="bottom-start" placement="bottom-start">
                                    <Button variant="outline" size="sm">BS</Button>
                                </Tooltip>
                            </Box>
                            <Box sx={{ gridColumn: "3", gridRow: "5", display: "flex", justifyContent: "center" }}>
                                <Tooltip title="bottom" placement="bottom">
                                    <Button variant="outline" size="sm">B</Button>
                                </Tooltip>
                            </Box>
                            <Box sx={{ gridColumn: "4", gridRow: "5", display: "flex", justifyContent: "center" }}>
                                <Tooltip title="bottom-end" placement="bottom-end">
                                    <Button variant="outline" size="sm">BE</Button>
                                </Tooltip>
                            </Box>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};
