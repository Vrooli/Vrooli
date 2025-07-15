import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { action } from "@storybook/addon-actions";
import {
    Box,
    Card,
    CardContent,
    Typography,
    TextField,
    FormControl,
    InputLabel,
    Select,
    MenuItem,
    FormControlLabel,
    Checkbox,
    Alert,
    Stack,
    Grid,
    useTheme,
} from "@mui/material";
import { Button } from "../../buttons/Button.js";
import { LinkDialog } from "./LinkDialog.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof LinkDialog> = {
    title: "Components/Dialogs/LinkDialog",
    component: LinkDialog,
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
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Showcase story with information about the component
export const Showcase: Story = {
    render: () => {
        const [mockData, setMockData] = useState({
            isOpen: false,
            isAdd: true,
            hasFromNode: false,
            hasToNode: false,
            fromNodeName: "Start Node",
            toNodeName: "End Node",
            language: "en",
            totalNodes: 5,
            existingLinks: 3,
        });

        const handleToggleDialog = () => {
            setMockData({ ...mockData, isOpen: !mockData.isOpen });
            if (!mockData.isOpen) {
                action("dialog-opened")();
            } else {
                action("dialog-closed")();
            }
        };

        return (
            <Stack spacing={3} sx={{ maxWidth: 600 }}>
                {/* Control Panel */}
                <Card>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            LinkDialog Controls
                        </Typography>
                    
                        <Stack spacing={2}>
                            <Box>
                                <FormControlLabel
                                    control={
                                        <Checkbox
                                            checked={mockData.isAdd}
                                            onChange={(e) => setMockData({ ...mockData, isAdd: e.target.checked })}
                                        />
                                    }
                                    label="Add Mode (vs Edit Mode)"
                                />
                                <Typography variant="caption" color="text.secondary" display="block">
                                    Determines if creating a new link or editing existing one
                                </Typography>
                            </Box>

                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={mockData.hasFromNode}
                                        onChange={(e) => setMockData({ ...mockData, hasFromNode: e.target.checked })}
                                    />
                                }
                                label="Pre-select From Node"
                            />

                            <FormControlLabel
                                control={
                                    <Checkbox
                                        checked={mockData.hasToNode}
                                        onChange={(e) => setMockData({ ...mockData, hasToNode: e.target.checked })}
                                    />
                                }
                                label="Pre-select To Node"
                            />

                            <TextField
                                label="From Node Name"
                                value={mockData.fromNodeName}
                                onChange={(e) => setMockData({ ...mockData, fromNodeName: e.target.value })}
                                fullWidth
                            />

                            <TextField
                                label="To Node Name"
                                value={mockData.toNodeName}
                                onChange={(e) => setMockData({ ...mockData, toNodeName: e.target.value })}
                                fullWidth
                            />

                            <FormControl fullWidth>
                                <InputLabel>Language</InputLabel>
                                <Select
                                    value={mockData.language}
                                    label="Language"
                                    onChange={(e) => setMockData({ ...mockData, language: e.target.value })}
                                >
                                    <MenuItem value="en">English</MenuItem>
                                    <MenuItem value="es">Spanish</MenuItem>
                                    <MenuItem value="fr">French</MenuItem>
                                    <MenuItem value="de">German</MenuItem>
                                </Select>
                            </FormControl>

                            <TextField
                                label="Total Nodes in Routine"
                                type="number"
                                inputProps={{ min: 2 }}
                                value={mockData.totalNodes}
                                onChange={(e) => setMockData({ ...mockData, totalNodes: parseInt(e.target.value) || 2 })}
                                fullWidth
                            />

                            <TextField
                                label="Existing Links"
                                type="number"
                                inputProps={{ min: 0 }}
                                value={mockData.existingLinks}
                                onChange={(e) => setMockData({ ...mockData, existingLinks: parseInt(e.target.value) || 0 })}
                                fullWidth
                            />
                        </Stack>
                    </CardContent>
                </Card>

                {/* Trigger Button */}
                <Box sx={{ textAlign: "center" }}>
                    <Button onClick={handleToggleDialog} variant="primary" size="lg">
                        {mockData.isOpen ? "Close" : "Open"} Link Dialog
                    </Button>
                </Box>

                {/* Info about link creation */}
                <Alert severity="success">
                    <Typography variant="body2">
                        <strong>Link Dialog Features:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li>Select From and To nodes with dropdown menus</li>
                        <li>Visual arrow indicating link direction</li>
                        <li>Validation to prevent invalid connections</li>
                        <li>Future: Conditional link creation with "when" clauses</li>
                        <li>Delete option for existing links (edit mode)</li>
                        <li>Prevents duplicate links and self-links</li>
                    </Box>
                </Alert>

                {/* Current state */}
                <Alert severity={mockData.isOpen ? "success" : "info"}>
                    <Typography variant="body2">
                        <strong>Current Configuration:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li>Mode: {mockData.isAdd ? "Add New Link" : "Edit Existing Link"}</li>
                        <li>From Node: {mockData.hasFromNode ? mockData.fromNodeName : "Not pre-selected"}</li>
                        <li>To Node: {mockData.hasToNode ? mockData.toNodeName : "Not pre-selected"}</li>
                        <li>Language: {mockData.language}</li>
                        <li>Available Nodes: {mockData.totalNodes}</li>
                        <li>Existing Links: {mockData.existingLinks}</li>
                        <li>Dialog State: {mockData.isOpen ? "Open" : "Closed"}</li>
                    </Box>
                </Alert>

                {/* Component status */}
                <Alert severity="warning">
                    <Typography variant="body2">
                        <strong>Component Status:</strong>
                    </Typography>
                    <Typography variant="body2" sx={{ mt: 1 }}>
                        This component is currently commented out in the source code. 
                        It would allow users to create and edit links between nodes in a routine graph,
                        specifying the execution flow and optional conditions.
                    </Typography>
                </Alert>

                {/* Dialog Component */}
                <LinkDialog />
            </Stack>
        );
    },
};

// Different modes demonstration
export const LinkModes: Story = {
    render: () => {
        const modes = [
            {
                title: "Add New Link",
                description: "Creating a new connection between two nodes",
                isAdd: true,
                color: "#e8f5e8",
                borderColor: "#4caf50",
            },
            {
                title: "Edit Existing Link",
                description: "Modifying or deleting an existing connection",
                isAdd: false,
                color: "#fff3e0",
                borderColor: "#ff9800",
            },
        ];

        return (
            <div style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
                <h3 style={{ textAlign: "center", marginBottom: "24px" }}>
                    Link Dialog Modes
                </h3>
                
                {modes.map((mode, i) => (
                    <div key={i} style={{
                        padding: "24px",
                        backgroundColor: mode.color,
                        borderRadius: "8px",
                        border: `1px solid ${mode.borderColor}`,
                    }}>
                        <h4 style={{ margin: "0 0 8px 0", color: "#333" }}>
                            {mode.title}
                        </h4>
                        <p style={{ margin: "0 0 16px 0", color: "#666" }}>
                            {mode.description}
                        </p>
                        
                        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "16px", fontSize: "14px" }}>
                            <div>
                                <strong>Features:</strong>
                                <ul style={{ marginTop: "4px", paddingLeft: "16px" }}>
                                    {mode.isAdd ? (
                                        <>
                                            <li>From/To node selection</li>
                                            <li>Duplicate prevention</li>
                                            <li>Validation checks</li>
                                        </>
                                    ) : (
                                        <>
                                            <li>Pre-filled node selections</li>
                                            <li>Delete option available</li>
                                            <li>Condition editing</li>
                                        </>
                                    )}
                                </ul>
                            </div>
                            <div>
                                <strong>Actions:</strong>
                                <ul style={{ marginTop: "4px", paddingLeft: "16px" }}>
                                    {mode.isAdd ? (
                                        <>
                                            <li>Create link button</li>
                                            <li>Cancel creation</li>
                                        </>
                                    ) : (
                                        <>
                                            <li>Save changes</li>
                                            <li>Delete link</li>
                                            <li>Cancel editing</li>
                                        </>
                                    )}
                                </ul>
                            </div>
                        </div>
                    </div>
                ))}
            </div>
        );
    },
};

// Future implementation preview
export const FutureImplementation: Story = {
    render: () => {
        return (
            <div style={{ display: "flex", flexDirection: "column", gap: "24px", alignItems: "center" }}>
                <div style={{
                    padding: "24px",
                    backgroundColor: "#e3f2fd",
                    borderRadius: "8px",
                    border: "1px solid #2196f3",
                    textAlign: "center",
                    maxWidth: "600px",
                }}>
                    <h3 style={{ marginTop: 0, color: "#1976d2" }}>Future Implementation</h3>
                    <p style={{ color: "#1976d2", margin: "0 0 16px 0" }}>
                        When implemented, the Link Dialog will provide:
                    </p>
                    
                    <div style={{
                        display: "grid",
                        gridTemplateColumns: "1fr 1fr",
                        gap: "16px",
                        textAlign: "left",
                        marginBottom: "16px",
                    }}>
                        <div>
                            <h4 style={{ color: "#1976d2", marginBottom: "8px" }}>Core Features:</h4>
                            <ul style={{ color: "#1976d2", marginBottom: 0, fontSize: "14px" }}>
                                <li>Node selection dropdowns</li>
                                <li>Visual link direction arrow</li>
                                <li>Duplicate link prevention</li>
                                <li>Self-link validation</li>
                            </ul>
                        </div>
                        <div>
                            <h4 style={{ color: "#1976d2", marginBottom: "8px" }}>Advanced Features:</h4>
                            <ul style={{ color: "#1976d2", marginBottom: 0, fontSize: "14px" }}>
                                <li>Conditional "when" clauses</li>
                                <li>Link operation settings</li>
                                <li>Multi-language support</li>
                                <li>Delete confirmation</li>
                            </ul>
                        </div>
                    </div>
                    
                    <div style={{
                        padding: "12px",
                        backgroundColor: "rgba(255, 255, 255, 0.5)",
                        borderRadius: "4px",
                        fontSize: "12px",
                        fontStyle: "italic",
                    }}>
                        Help text: "This dialog allows you create new links between nodes, which specifies 
                        the order in which the nodes are executed. In the future, links will also be able 
                        to specify conditions, which must be true in order for the path to be available."
                    </div>
                </div>

                <div style={{
                    display: "flex",
                    gap: "16px",
                    padding: "16px",
                    backgroundColor: "#f5f5f5",
                    borderRadius: "8px",
                    fontSize: "14px",
                }}>
                    <div style={{ textAlign: "center" }}>
                        <div style={{
                            width: "60px",
                            height: "40px",
                            backgroundColor: "#4caf50",
                            borderRadius: "4px",
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            color: "white",
                            fontWeight: "bold",
                            marginBottom: "4px",
                        }}>
                            FROM
                        </div>
                        <small>Source Node</small>
                    </div>
                    
                    <div style={{
                        display: "flex",
                        alignItems: "center",
                        fontSize: "24px",
                        color: "#666",
                    }}>
                        ⮕
                    </div>
                    
                    <div style={{ textAlign: "center" }}>
                        <div style={{
                            width: "60px",
                            height: "40px",
                            backgroundColor: "#2196f3",
                            borderRadius: "4px",
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            color: "white",
                            fontWeight: "bold",
                            marginBottom: "4px",
                        }}>
                            TO
                        </div>
                        <small>Target Node</small>
                    </div>
                </div>
            </div>
        );
    },
};
