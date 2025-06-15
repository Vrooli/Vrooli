import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import RadioGroup from "@mui/material/RadioGroup";
import FormControlLabel from "@mui/material/FormControlLabel";
import Radio from "@mui/material/Radio";
import Switch from "@mui/material/Switch";
import TextField from "@mui/material/TextField";
import Slider from "@mui/material/Slider";
import Paper from "@mui/material/Paper";
import { LineGraph } from "./LineGraph.js";

const meta: Meta<typeof LineGraph> = {
    title: "Components/Graphs/LineGraph",
    component: LineGraph,
    parameters: {
        layout: "fullscreen",
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Generate sample data sets
const generateSampleData = (type: "increasing" | "decreasing" | "random" | "sine" | "flat", length: number = 20) => {
    const data: number[] = [];
    
    for (let i = 0; i < length; i++) {
        switch (type) {
            case "increasing":
                data.push(10 + i * 2 + Math.random() * 5);
                break;
            case "decreasing":
                data.push(50 - i * 1.5 + Math.random() * 3);
                break;
            case "random":
                data.push(Math.random() * 40 + 10);
                break;
            case "sine":
                data.push(25 + Math.sin(i * 0.5) * 15 + Math.random() * 3);
                break;
            case "flat":
                data.push(20 + Math.random() * 2);
                break;
            default:
                data.push(Math.random() * 30 + 10);
        }
    }
    
    return data;
};

// Generate labeled data
const generateLabeledData = (type: "increasing" | "decreasing" | "random" | "sine" | "flat", length: number = 12) => {
    const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
    const hours = Array.from({ length: 24 }, (_, i) => `${i}:00`);
    const days = Array.from({ length: 7 }, (_, i) => ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"][i]);
    
    const labels = length <= 7 ? days.slice(0, length) : 
                  length <= 12 ? months.slice(0, length) :
                  length <= 24 ? hours.slice(0, length) :
                  Array.from({ length }, (_, i) => `Point ${i + 1}`);
    
    return labels.map((label, i) => {
        let value: number;
        switch (type) {
            case "increasing":
                value = 10 + i * 2 + Math.random() * 5;
                break;
            case "decreasing":
                value = 50 - i * 1.5 + Math.random() * 3;
                break;
            case "random":
                value = Math.random() * 40 + 10;
                break;
            case "sine":
                value = 25 + Math.sin(i * 0.5) * 15 + Math.random() * 3;
                break;
            case "flat":
                value = 20 + Math.random() * 2;
                break;
            default:
                value = Math.random() * 30 + 10;
        }
        return { label, value: Math.round(value * 10) / 10 };
    });
};

// Interactive LineGraph Playground
export const LineGraphShowcase: Story = {
    render: () => {
        // State for controls
        const [width, setWidth] = useState(400);
        const [height, setHeight] = useState(250);
        const [lineColor, setLineColor] = useState("#1976d2");
        const [dotColor, setDotColor] = useState("#1976d2");
        const [lineWidth, setLineWidth] = useState(2);
        const [hideAxes, setHideAxes] = useState(false);
        const [hideTooltips, setHideTooltips] = useState(false);
        const [dataType, setDataType] = useState<"increasing" | "decreasing" | "random" | "sine" | "flat">("increasing");
        const [dataLength, setDataLength] = useState(12);
        const [useLabels, setUseLabels] = useState(true);

        // Generate data based on current settings
        const data = useLabels 
            ? generateLabeledData(dataType, dataLength)
            : generateSampleData(dataType, dataLength);

        const presetColors = [
            { name: "Blue", value: "#1976d2" },
            { name: "Green", value: "#388e3c" },
            { name: "Red", value: "#d32f2f" },
            { name: "Purple", value: "#7b1fa2" },
            { name: "Orange", value: "#f57c00" },
            { name: "Teal", value: "#00796b" },
        ];

        return (
            <Box sx={{ 
                p: 2, 
                height: "100vh", 
                overflow: "auto",
                bgcolor: "background.default" 
            }}>
                <Box sx={{ 
                    display: "flex", 
                    gap: 2, 
                    flexDirection: "column",
                    maxWidth: 1400, 
                    mx: "auto" 
                }}>
                    {/* Controls Section */}
                    <Paper sx={{ 
                        p: 3, 
                        borderRadius: 2, 
                        boxShadow: 1,
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>LineGraph Controls</Typography>
                        
                        <Box sx={{ 
                            display: "grid", 
                            gridTemplateColumns: { xs: "1fr", sm: "repeat(2, 1fr)", md: "repeat(3, 1fr)", lg: "repeat(4, 1fr)" },
                            gap: 3 
                        }}>
                            {/* Dimensions */}
                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Width: {width}px</Typography>
                                <Slider
                                    value={width}
                                    onChange={(_, value) => setWidth(value as number)}
                                    min={200}
                                    max={800}
                                    step={10}
                                    size="small"
                                />
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Height: {height}px</Typography>
                                <Slider
                                    value={height}
                                    onChange={(_, value) => setHeight(value as number)}
                                    min={100}
                                    max={500}
                                    step={10}
                                    size="small"
                                />
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Line Width: {lineWidth}px</Typography>
                                <Slider
                                    value={lineWidth}
                                    onChange={(_, value) => setLineWidth(value as number)}
                                    min={1}
                                    max={8}
                                    step={0.5}
                                    size="small"
                                />
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Data Points: {dataLength}</Typography>
                                <Slider
                                    value={dataLength}
                                    onChange={(_, value) => setDataLength(value as number)}
                                    min={3}
                                    max={50}
                                    step={1}
                                    size="small"
                                />
                            </Box>

                            {/* Data Type */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Data Pattern</FormLabel>
                                <RadioGroup
                                    value={dataType}
                                    onChange={(e) => setDataType(e.target.value as typeof dataType)}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="increasing" control={<Radio size="small" />} label="Increasing" sx={{ m: 0 }} />
                                    <FormControlLabel value="decreasing" control={<Radio size="small" />} label="Decreasing" sx={{ m: 0 }} />
                                    <FormControlLabel value="random" control={<Radio size="small" />} label="Random" sx={{ m: 0 }} />
                                    <FormControlLabel value="sine" control={<Radio size="small" />} label="Sine Wave" sx={{ m: 0 }} />
                                    <FormControlLabel value="flat" control={<Radio size="small" />} label="Flat" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>

                            {/* Color Presets */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Line Color</FormLabel>
                                <RadioGroup
                                    value={lineColor}
                                    onChange={(e) => {
                                        setLineColor(e.target.value);
                                        setDotColor(e.target.value); // Sync dot color
                                    }}
                                    sx={{ gap: 0.5 }}
                                >
                                    {presetColors.map((color) => (
                                        <FormControlLabel 
                                            key={color.name}
                                            value={color.value} 
                                            control={<Radio size="small" sx={{ color: color.value, "&.Mui-checked": { color: color.value } }} />} 
                                            label={color.name} 
                                            sx={{ m: 0 }} 
                                        />
                                    ))}
                                </RadioGroup>
                            </FormControl>

                            {/* Custom Colors */}
                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Custom Line Color</Typography>
                                <TextField
                                    type="color"
                                    value={lineColor}
                                    onChange={(e) => {
                                        setLineColor(e.target.value);
                                        setDotColor(e.target.value); // Sync dot color
                                    }}
                                    size="small"
                                    sx={{ width: "100%" }}
                                />
                            </Box>

                            <Box>
                                <Typography variant="subtitle2" sx={{ mb: 1 }}>Custom Dot Color</Typography>
                                <TextField
                                    type="color"
                                    value={dotColor}
                                    onChange={(e) => setDotColor(e.target.value)}
                                    size="small"
                                    sx={{ width: "100%" }}
                                />
                            </Box>

                            {/* Toggle Options */}
                            <Box>
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Options</FormLabel>
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
                                    <FormControlLabel
                                        control={
                                            <Switch
                                                checked={hideAxes}
                                                onChange={(e) => setHideAxes(e.target.checked)}
                                                size="small"
                                            />
                                        }
                                        label="Hide Axes"
                                    />
                                    <FormControlLabel
                                        control={
                                            <Switch
                                                checked={hideTooltips}
                                                onChange={(e) => setHideTooltips(e.target.checked)}
                                                size="small"
                                            />
                                        }
                                        label="Hide Tooltips"
                                    />
                                    <FormControlLabel
                                        control={
                                            <Switch
                                                checked={useLabels}
                                                onChange={(e) => setUseLabels(e.target.checked)}
                                                size="small"
                                            />
                                        }
                                        label="Use Labels"
                                    />
                                </Box>
                            </Box>
                        </Box>
                    </Paper>

                    {/* Graph Display */}
                    <Paper sx={{ 
                        p: 3, 
                        borderRadius: 2, 
                        boxShadow: 1,
                        display: "flex",
                        flexDirection: "column",
                        alignItems: "center"
                    }}>
                        <Typography variant="h6" sx={{ mb: 2 }}>LineGraph Preview</Typography>
                        
                        {/* Graph Container */}
                        <Box sx={{ 
                            border: "1px solid",
                            borderColor: "divider",
                            borderRadius: 1,
                            p: 2,
                            bgcolor: "background.default"
                        }}>
                            <LineGraph
                                data={data}
                                dims={{ width, height }}
                                lineColor={lineColor}
                                dotColor={dotColor}
                                lineWidth={lineWidth}
                                hideAxes={hideAxes}
                                hideTooltips={hideTooltips}
                            />
                        </Box>

                        {/* Data Preview */}
                        <Box sx={{ mt: 3, width: "100%", maxWidth: 600 }}>
                            <Typography variant="subtitle2" sx={{ mb: 1 }}>Current Data ({data.length} points):</Typography>
                            <Paper sx={{ 
                                p: 2, 
                                bgcolor: "grey.50", 
                                maxHeight: 150, 
                                overflow: "auto",
                                fontSize: "0.75rem",
                                fontFamily: "monospace"
                            }}>
                                {useLabels ? (
                                    data.map((point, i) => (
                                        <div key={i}>
                                            {typeof point === "object" ? `${point.label}: ${point.value}` : point}
                                        </div>
                                    ))
                                ) : (
                                    <div>[{data.join(", ")}]</div>
                                )}
                            </Paper>
                        </Box>
                    </Paper>
                </Box>
            </Box>
        );
    },
};

// Simple examples for different use cases
export const SimpleExample: Story = {
    args: {
        data: [10, 15, 8, 22, 18, 25, 12, 30, 28, 35],
        dims: { width: 400, height: 200 },
        lineColor: "#1976d2",
        lineWidth: 2,
    },
};

export const WithLabels: Story = {
    args: {
        data: [
            { label: "Jan", value: 120 },
            { label: "Feb", value: 135 },
            { label: "Mar", value: 108 },
            { label: "Apr", value: 142 },
            { label: "May", value: 158 },
            { label: "Jun", value: 165 },
        ],
        dims: { width: 400, height: 200 },
        lineColor: "#388e3c",
        lineWidth: 2,
    },
};

export const MiniChart: Story = {
    args: {
        data: generateSampleData("increasing", 15),
        dims: { width: 200, height: 60 },
        lineColor: "#7b1fa2",
        dotColor: "#7b1fa2",
        lineWidth: 3,
        hideAxes: true,
        hideTooltips: false,
    },
};

export const NoInteraction: Story = {
    args: {
        data: generateSampleData("sine", 20),
        dims: { width: 300, height: 150 },
        lineColor: "#d32f2f",
        dotColor: "#d32f2f",
        lineWidth: 2,
        hideAxes: true,
        hideTooltips: true,
    },
};

export const LargeDataSet: Story = {
    args: {
        data: generateSampleData("random", 100),
        dims: { width: 600, height: 300 },
        lineColor: "#00796b",
        dotColor: "#00796b",
        lineWidth: 1,
        hideAxes: false,
        hideTooltips: false,
    },
};