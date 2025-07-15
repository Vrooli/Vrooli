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
} from "@mui/material";
import { Button } from "../../buttons/Button.js";
import { SubroutineInfoDialog } from "./SubroutineInfoDialog.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof SubroutineInfoDialog> = {
    title: "Components/Dialogs/SubroutineInfoDialog",
    component: SubroutineInfoDialog,
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
            hasData: false,
            isEditing: false,
            numSubroutines: 3,
            subroutineIndex: 1,
            subroutineTitle: "Data Processing Step",
            subroutineType: "Data",
        });

        return (
            <Stack spacing={3} sx={{ maxWidth: 600 }}>
                {/* Control Panel */}
                <Card>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            SubroutineInfoDialog Controls
                        </Typography>
                        
                        <Stack spacing={2}>
                            <Box>
                                <FormControlLabel
                                    control={
                                        <Checkbox
                                            checked={mockData.hasData}
                                            onChange={(e) => setMockData({ ...mockData, hasData: e.target.checked })}
                                        />
                                    }
                                    label="Has Subroutine Data"
                                />
                                <Typography variant="caption" color="text.secondary" display="block">
                                    Simulates whether subroutine data is available
                                </Typography>
                            </Box>

                            <Box>
                                <FormControlLabel
                                    control={
                                        <Checkbox
                                            checked={mockData.isEditing}
                                            onChange={(e) => setMockData({ ...mockData, isEditing: e.target.checked })}
                                        />
                                    }
                                    label="Editing Mode"
                                />
                                <Typography variant="caption" color="text.secondary" display="block">
                                    Enables editing of subroutine properties
                                </Typography>
                            </Box>

                            <TextField
                                label="Number of Subroutines"
                                type="number"
                                inputProps={{ min: 1 }}
                                value={mockData.numSubroutines}
                                onChange={(e) => setMockData({ ...mockData, numSubroutines: parseInt(e.target.value) || 1 })}
                                fullWidth
                            />

                            <TextField
                                label="Subroutine Index"
                                type="number"
                                inputProps={{ min: 0, max: mockData.numSubroutines - 1 }}
                                value={mockData.subroutineIndex}
                                onChange={(e) => setMockData({ ...mockData, subroutineIndex: parseInt(e.target.value) || 0 })}
                                fullWidth
                            />

                            <TextField
                                label="Subroutine Title"
                                value={mockData.subroutineTitle}
                                onChange={(e) => setMockData({ ...mockData, subroutineTitle: e.target.value })}
                                fullWidth
                            />

                            <FormControl fullWidth>
                                <InputLabel>Subroutine Type</InputLabel>
                                <Select
                                    value={mockData.subroutineType}
                                    label="Subroutine Type"
                                    onChange={(e) => setMockData({ ...mockData, subroutineType: e.target.value })}
                                >
                                    <MenuItem value="Prompt">Prompt</MenuItem>
                                    <MenuItem value="Data">Data</MenuItem>
                                    <MenuItem value="Generate">Generate</MenuItem>
                                    <MenuItem value="Api">API</MenuItem>
                                    <MenuItem value="SmartContract">Smart Contract</MenuItem>
                                    <MenuItem value="WebContent">Web Content</MenuItem>
                                    <MenuItem value="Code">Code</MenuItem>
                                </Select>
                            </FormControl>
                        </Stack>
                    </CardContent>
                </Card>

                {/* Info about current state */}
                <Alert severity="info">
                    <Typography variant="body2">
                        <strong>Current State:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li>Has Data: {mockData.hasData ? "Yes" : "No"}</li>
                        <li>Editing Mode: {mockData.isEditing ? "Enabled" : "Disabled"}</li>
                        <li>Position: {mockData.subroutineIndex + 1} of {mockData.numSubroutines}</li>
                        <li>Title: {mockData.subroutineTitle}</li>
                        <li>Type: {mockData.subroutineType}</li>
                    </Box>
                </Alert>

                {/* Component status */}
                <Alert severity="warning">
                    <Typography variant="body2">
                        <strong>Component Status:</strong>
                    </Typography>
                    <Typography variant="body2" sx={{ mt: 1 }}>
                        This component is currently commented out in the source code. 
                        It would display detailed information about a subroutine in a routine, 
                        including its position, configuration, and type-specific settings.
                    </Typography>
                </Alert>

                {/* Dialog Component */}
                <SubroutineInfoDialog />
            </Stack>
        );
    },
};

// Placeholder for when component is implemented
export const WhenImplemented: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);

        const handleOpen = () => {
            setIsOpen(true);
        };

        const handleClose = () => {
            action("dialog-closed")();
            setIsOpen(false);
        };

        return (
            <Stack spacing={3} alignItems="center">
                <Alert severity="success" sx={{ textAlign: "center", maxWidth: 500 }}>
                    <Typography variant="h6" gutterBottom>
                        Future Implementation
                    </Typography>
                    <Typography variant="body2" sx={{ mb: 1 }}>
                        When implemented, this dialog will show:
                    </Typography>
                    <Box component="ul" sx={{ textAlign: "left", mb: 0, pl: 2 }}>
                        <li>Subroutine position and reordering controls</li>
                        <li>Name and description editing</li>
                        <li>Type-specific configuration forms</li>
                        <li>Resource lists and dependencies</li>
                        <li>Instruction editing</li>
                        <li>Language selection</li>
                        <li>Action buttons for save/cancel</li>
                    </Box>
                </Alert>

                <Button onClick={handleOpen} variant="primary" size="lg">
                    Open Subroutine Info Dialog (Future)
                </Button>

                <Typography variant="body2" color="text.secondary" sx={{ textAlign: "center", fontStyle: "italic" }}>
                    Dialog state: {isOpen ? "Would be open" : "Closed"}
                </Typography>
            </Stack>
        );
    },
};

// Mock data scenarios
export const MockDataScenarios: Story = {
    render: () => {
        const scenarios = [
            {
                title: "First Subroutine",
                description: "Position 1 of 5, Data type, can be edited",
                index: 0,
                total: 5,
                type: "Data",
                canEdit: true,
            },
            {
                title: "Middle Subroutine",
                description: "Position 3 of 5, API type, read-only",
                index: 2,
                total: 5,
                type: "Api",
                canEdit: false,
            },
            {
                title: "Last Subroutine",
                description: "Position 5 of 5, Generate type, can be edited",
                index: 4,
                total: 5,
                type: "Generate",
                canEdit: true,
            },
        ];

        return (
            <Stack spacing={2}>
                <Typography variant="h5" sx={{ textAlign: "center", mb: 3 }}>
                    Subroutine Info Dialog Scenarios
                </Typography>
                
                {scenarios.map((scenario, i) => (
                    <Card key={i}>
                        <CardContent>
                            <Typography variant="h6" gutterBottom>
                                {scenario.title}
                            </Typography>
                            <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
                                {scenario.description}
                            </Typography>
                            <Grid container spacing={1}>
                                <Grid item xs={4}>
                                    <Typography variant="caption">
                                        <strong>Position:</strong> {scenario.index + 1} of {scenario.total}
                                    </Typography>
                                </Grid>
                                <Grid item xs={4}>
                                    <Typography variant="caption">
                                        <strong>Type:</strong> {scenario.type}
                                    </Typography>
                                </Grid>
                                <Grid item xs={4}>
                                    <Typography variant="caption">
                                        <strong>Editable:</strong> {scenario.canEdit ? "Yes" : "No"}
                                    </Typography>
                                </Grid>
                            </Grid>
                        </CardContent>
                    </Card>
                ))}
                
                <Alert severity="info" sx={{ textAlign: "center" }}>
                    <Typography variant="body2" fontStyle="italic">
                        These scenarios show what the dialog would display when implemented
                    </Typography>
                </Alert>
            </Stack>
        );
    },
};
