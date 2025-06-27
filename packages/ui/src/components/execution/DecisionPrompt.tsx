import { Box, Button, Radio, RadioGroup, FormControlLabel, Typography, Paper } from "@mui/material";
import { useState } from "react";
import { type DeferredDecisionData } from "./RoutineExecutor.js";

interface DecisionPromptProps {
    decision: DeferredDecisionData;
    onSubmit: (decision: unknown) => void;
    onCancel: () => void;
}

export function DecisionPrompt({
    decision,
    onSubmit,
    onCancel,
}: DecisionPromptProps) {
    const [selectedOption, setSelectedOption] = useState<string>("");

    const handleSubmit = () => {
        if (selectedOption) {
            onSubmit({ optionId: selectedOption });
        }
    };

    return (
        <Paper elevation={3} sx={{ p: 3, maxWidth: 600, mx: "auto" }}>
            <Typography variant="h6" gutterBottom>
                Decision Required
            </Typography>
            
            <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
                {decision.prompt}
            </Typography>

            <RadioGroup
                value={selectedOption}
                onChange={(e) => setSelectedOption(e.target.value)}
                sx={{ mb: 3 }}
            >
                {decision.options.map((option) => (
                    <FormControlLabel
                        key={option.id}
                        value={option.id}
                        control={<Radio />}
                        label={option.label}
                        sx={{
                            mb: 1,
                            "& .MuiFormControlLabel-label": {
                                flex: 1,
                            },
                        }}
                    />
                ))}
            </RadioGroup>

            <Box sx={{ display: "flex", gap: 2, justifyContent: "flex-end" }}>
                <Button variant="outlined" onClick={onCancel}>
                    Cancel
                </Button>
                <Button
                    variant="contained"
                    onClick={handleSubmit}
                    disabled={!selectedOption}
                >
                    Submit Decision
                </Button>
            </Box>
        </Paper>
    );
}
