import { Box, Typography, Paper, Divider, LinearProgress, Chip, Alert, Table, TableBody, TableCell, TableRow, Collapse, useTheme } from "@mui/material";
import { IconButton } from "../buttons/IconButton.js";
import { type ExecutionStep } from "./RoutineExecutor.js";
import { IconCommon } from "../../icons/Icons.js";
import { type ResourceVersion, ResourceSubType, CodeLanguage } from "@vrooli/shared";
import { useState } from "react";
import { CodeInputBase } from "../inputs/CodeInput/CodeInput.js";

interface StepDetailsProps {
    step?: ExecutionStep;
    contextValues: Record<string, unknown>;
    totalSteps: number;
    currentStepIndex: number;
    resourceVersion?: ResourceVersion;
    currentSubroutine?: ResourceVersion;
    onConfigExpandChange?: (expanded: boolean) => void;
}

function formatValue(value: unknown): string {
    if (value === null || value === undefined) return "—";
    if (typeof value === "object") return JSON.stringify(value, null, 2);
    return String(value);
}

function calculateDuration(startTime?: string, endTime?: string): string {
    if (!startTime) return "—";
    const start = new Date(startTime).getTime();
    const end = endTime ? new Date(endTime).getTime() : Date.now();
    const duration = Math.floor((end - start) / 1000);
    
    if (duration < 60) return `${duration}s`;
    if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`;
    return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`;
}

// Component to display routine-type specific information
function RoutineTypeDetails({ resourceVersion, config, onConfigExpandChange }: { 
    resourceVersion?: ResourceVersion; 
    config?: any;
    onConfigExpandChange?: (expanded: boolean) => void;
}) {
    const theme = useTheme();
    const [showFullConfig, setShowFullConfig] = useState(false); // Default to collapsed
    
    const handleToggleConfig = () => {
        const newState = !showFullConfig;
        setShowFullConfig(newState);
        onConfigExpandChange?.(newState);
    };
    
    if (!resourceVersion || !resourceVersion.resourceSubType) return null;

    const routineType = resourceVersion.resourceSubType;
    const routineConfig = config || {};

    const typeDescriptions: Record<ResourceSubType, string> = {
        [ResourceSubType.RoutineMultiStep]: "Complex workflow with multiple steps and branches",
        [ResourceSubType.RoutineInternalAction]: "Internal Vrooli action using MCP tools",
        [ResourceSubType.RoutineApi]: "External API call",
        [ResourceSubType.RoutineCode]: "Code execution in sandboxed environment",
        [ResourceSubType.RoutineData]: "Data-only routine with outputs",
        [ResourceSubType.RoutineGenerate]: "AI/LLM content generation",
        [ResourceSubType.RoutineInformational]: "Information collection via forms",
        [ResourceSubType.RoutineSmartContract]: "Blockchain smart contract interaction",
        [ResourceSubType.RoutineWeb]: "Web search and scraping",
    };

    const renderTypeSpecificContent = () => {
        switch (routineType) {
            case ResourceSubType.RoutineApi:
                const apiConfig = routineConfig.callDataApi || {};
                return (
                    <>
                        {apiConfig.endpoint && (
                            <Box>
                                <Typography variant="caption" color="text.secondary">Endpoint</Typography>
                                <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                                    {apiConfig.method || "GET"} {apiConfig.endpoint}
                                </Typography>
                            </Box>
                        )}
                        {apiConfig.headers && Object.keys(apiConfig.headers).length > 0 && (
                            <Box sx={{ mt: 2 }}>
                                <Typography variant="caption" color="text.secondary">Headers</Typography>
                                <Table size="small" sx={{ mt: 0.5 }}>
                                    <TableBody>
                                        {Object.entries(apiConfig.headers).map(([key, value]) => (
                                            <TableRow key={key}>
                                                <TableCell sx={{ border: "none", py: 0.5, fontFamily: "monospace" }}>
                                                    {key}
                                                </TableCell>
                                                <TableCell sx={{ border: "none", py: 0.5, fontFamily: "monospace" }}>
                                                    {String(value)}
                                                </TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            </Box>
                        )}
                    </>
                );

            case ResourceSubType.RoutineGenerate:
                const generateConfig = routineConfig.callDataGenerate || {};
                return (
                    <>
                        {generateConfig.model && (
                            <Box>
                                <Typography variant="caption" color="text.secondary">AI Model</Typography>
                                <Typography variant="body2">{generateConfig.model}</Typography>
                            </Box>
                        )}
                        {generateConfig.prompt && (
                            <Box sx={{ mt: 2 }}>
                                <Typography variant="caption" color="text.secondary">Prompt Template</Typography>
                                <Typography 
                                    variant="body2" 
                                    sx={{ 
                                        mt: 0.5, 
                                        p: 1, 
                                        bgcolor: "action.hover", 
                                        borderRadius: 1,
                                        fontFamily: "monospace",
                                        whiteSpace: "pre-wrap",
                                    }}
                                >
                                    {generateConfig.prompt}
                                </Typography>
                            </Box>
                        )}
                    </>
                );

            case ResourceSubType.RoutineCode:
                const codeConfig = routineConfig.callDataCode || {};
                return (
                    <>
                        {resourceVersion.codeLanguage && (
                            <Box>
                                <Typography variant="caption" color="text.secondary">Language</Typography>
                                <Typography variant="body2">{resourceVersion.codeLanguage}</Typography>
                            </Box>
                        )}
                        {codeConfig.code && (
                            <Box sx={{ mt: 2 }}>
                                <Typography variant="caption" color="text.secondary">Code</Typography>
                                <Typography 
                                    variant="body2" 
                                    component="pre"
                                    sx={{ 
                                        mt: 0.5, 
                                        p: 1, 
                                        bgcolor: "action.hover", 
                                        borderRadius: 1,
                                        fontFamily: "monospace",
                                        whiteSpace: "pre-wrap",
                                        overflow: "auto",
                                    }}
                                >
                                    {codeConfig.code}
                                </Typography>
                            </Box>
                        )}
                    </>
                );

            case ResourceSubType.RoutineMultiStep:
                const graphConfig = routineConfig.graph || {};
                const nodeCount = graphConfig.nodes?.length || 0;
                const edgeCount = graphConfig.edges?.length || 0;
                return (
                    <>
                        <Box sx={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 2 }}>
                            <Box>
                                <Typography variant="caption" color="text.secondary">Total Nodes</Typography>
                                <Typography variant="body2">{nodeCount}</Typography>
                            </Box>
                            <Box>
                                <Typography variant="caption" color="text.secondary">Connections</Typography>
                                <Typography variant="body2">{edgeCount}</Typography>
                            </Box>
                        </Box>
                        {graphConfig.nodes && graphConfig.nodes.length > 0 && (
                            <Box sx={{ mt: 2 }}>
                                <Typography variant="caption" color="text.secondary">Subroutines</Typography>
                                <Box sx={{ mt: 0.5, display: "flex", flexWrap: "wrap", gap: 0.5 }}>
                                    {graphConfig.nodes
                                        .filter((node: any) => node.data?.routine?.name)
                                        .map((node: any, index: number) => (
                                            <Chip
                                                key={node.id || index}
                                                label={node.data.routine.name}
                                                size="small"
                                                variant="outlined"
                                                sx={{ fontFamily: "monospace" }}
                                            />
                                        ))}
                                </Box>
                            </Box>
                        )}
                    </>
                );

            case ResourceSubType.RoutineWeb:
                const webConfig = routineConfig.callDataWeb || {};
                return (
                    <>
                        {webConfig.searchQuery && (
                            <Box>
                                <Typography variant="caption" color="text.secondary">Search Query</Typography>
                                <Typography variant="body2">{webConfig.searchQuery}</Typography>
                            </Box>
                        )}
                        {webConfig.urls && webConfig.urls.length > 0 && (
                            <Box sx={{ mt: 2 }}>
                                <Typography variant="caption" color="text.secondary">Target URLs</Typography>
                                <Box sx={{ mt: 0.5 }}>
                                    {webConfig.urls.map((url: string, index: number) => (
                                        <Typography key={index} variant="body2" sx={{ fontFamily: "monospace" }}>
                                            {url}
                                        </Typography>
                                    ))}
                                </Box>
                            </Box>
                        )}
                    </>
                );

            default:
                return (
                    <Typography variant="body2" color="text.secondary">
                        Configuration details for this routine type
                    </Typography>
                );
        }
    };

    return (
        <Box sx={{ minHeight: "100%", display: "flex", flexDirection: "column" }}>
            <Box sx={{ flex: 1 }}>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    {typeDescriptions[routineType]}
                </Typography>

                {renderTypeSpecificContent()}

                {/* Expandable Full Configuration */}
                <Collapse in={showFullConfig} timeout="auto" unmountOnExit>
                    <Box sx={{ mt: 2 }}>
                        <Divider sx={{ mb: 2 }} />
                        <Typography variant="caption" color="text.secondary" sx={{ display: "block", mb: 1 }}>
                            Full Configuration
                        </Typography>
                        <CodeInputBase
                            codeLanguage={CodeLanguage.Json}
                            content={JSON.stringify(routineConfig, null, 2)}
                            disabled={true}
                            handleCodeLanguageChange={() => { }}
                            handleContentChange={() => { }}
                            name="routine-config"
                        />
                    </Box>
                </Collapse>
            </Box>

            {/* Expand/Collapse button sticky at bottom */}
            <Box
                sx={{
                    position: "sticky",
                    bottom: 0,
                    display: "flex",
                    justifyContent: "flex-end",
                    pt: 2,
                    pb: 1,
                    mt: 2,
                    zIndex: 100,
                }}
            >
                <IconButton
                    size="small"
                    onClick={handleToggleConfig}
                    title={showFullConfig ? "Compress configuration" : "Show full configuration"}
                    sx={{
                        bgcolor: "background.paper",
                        boxShadow: 2,
                        border: 1,
                        borderColor: "divider",
                        "&:hover": {
                            bgcolor: "action.hover",
                            boxShadow: 3,
                        },
                    }}
                >
                    <IconCommon name={showFullConfig ? "Compress" : "OpenThread"} size={20} />
                </IconButton>
            </Box>
        </Box>
    );
}

export function StepDetails({
    step,
    contextValues,
    totalSteps,
    currentStepIndex,
    resourceVersion,
    currentSubroutine,
    onConfigExpandChange,
}: StepDetailsProps) {
    // For single-step routines, show just the routine information
    if (!step && totalSteps === 0) {
        return (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                {/* Routine-Specific Information */}
                {resourceVersion && (
                    <RoutineTypeDetails 
                        resourceVersion={resourceVersion}
                        config={resourceVersion.config}
                        onConfigExpandChange={onConfigExpandChange}
                    />
                )}

                {/* Context Values */}
                {Object.keys(contextValues).length > 0 && (
                    <Paper variant="outlined" sx={{ p: 2 }}>
                        <Typography variant="subtitle2" gutterBottom>
                            Context
                        </Typography>
                        <Box sx={{ mt: 1 }}>
                            {Object.entries(contextValues).map(([key, value], index) => (
                                <Box key={key}>
                                    {index > 0 && <Divider sx={{ my: 1 }} />}
                                    <Box>
                                        <Typography variant="caption" color="text.secondary">
                                            {key}
                                        </Typography>
                                        <Typography
                                            variant="body2"
                                            sx={{
                                                fontFamily: "monospace",
                                                whiteSpace: "pre-wrap",
                                                wordBreak: "break-word",
                                            }}
                                        >
                                            {formatValue(value)}
                                        </Typography>
                                    </Box>
                                </Box>
                            ))}
                        </Box>
                    </Paper>
                )}
            </Box>
        );
    }

    if (!step) {
        return (
            <Box sx={{ p: 3, textAlign: "center" }}>
                <Typography color="text.secondary">
                    No step selected
                </Typography>
            </Box>
        );
    }


    return (
        <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>

            {/* Error Alert */}
            {step.error && (
                <Alert severity="error" icon={<IconCommon name="Error" size={20} />}>
                    <Typography variant="subtitle2" gutterBottom>
                        Error Details
                    </Typography>
                    <Typography variant="body2">
                        {step.error}
                    </Typography>
                    {step.errorDetails && (
                        <Typography variant="caption" component="pre" sx={{ mt: 1, whiteSpace: "pre-wrap" }}>
                            {step.errorDetails}
                        </Typography>
                    )}
                </Alert>
            )}

            {/* Routine-Specific Information */}
            {(resourceVersion || currentSubroutine) && (
                <RoutineTypeDetails 
                    resourceVersion={currentSubroutine || resourceVersion}
                    config={(currentSubroutine || resourceVersion)?.config}
                    onConfigExpandChange={onConfigExpandChange}
                />
            )}

            {/* Inputs */}
            {step.inputs && Object.keys(step.inputs).length > 0 && (
                <Paper variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="subtitle2" gutterBottom>
                        Inputs
                    </Typography>
                    <Box sx={{ mt: 1 }}>
                        {Object.entries(step.inputs).map(([key, value], index) => (
                            <Box key={key}>
                                {index > 0 && <Divider sx={{ my: 1 }} />}
                                <Box>
                                    <Typography variant="caption" color="text.secondary">
                                        {key}
                                    </Typography>
                                    <Typography
                                        variant="body2"
                                        sx={{
                                            fontFamily: "monospace",
                                            whiteSpace: "pre-wrap",
                                            wordBreak: "break-word",
                                        }}
                                    >
                                        {formatValue(value)}
                                    </Typography>
                                </Box>
                            </Box>
                        ))}
                    </Box>
                </Paper>
            )}

            {/* Outputs */}
            {step.outputs && Object.keys(step.outputs).length > 0 && (
                <Paper variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="subtitle2" gutterBottom>
                        Outputs
                    </Typography>
                    <Box sx={{ mt: 1 }}>
                        {Object.entries(step.outputs).map(([key, value], index) => (
                            <Box key={key}>
                                {index > 0 && <Divider sx={{ my: 1 }} />}
                                <Box>
                                    <Typography variant="caption" color="text.secondary">
                                        {key}
                                    </Typography>
                                    <Typography
                                        variant="body2"
                                        sx={{
                                            fontFamily: "monospace",
                                            whiteSpace: "pre-wrap",
                                            wordBreak: "break-word",
                                        }}
                                    >
                                        {formatValue(value)}
                                    </Typography>
                                </Box>
                            </Box>
                        ))}
                    </Box>
                </Paper>
            )}

            {/* Context Values */}
            {Object.keys(contextValues).length > 0 && (
                <Paper variant="outlined" sx={{ p: 2 }}>
                    <Typography variant="subtitle2" gutterBottom>
                        Context
                    </Typography>
                    <Box sx={{ mt: 1 }}>
                        {Object.entries(contextValues).map(([key, value], index) => (
                            <Box key={key}>
                                {index > 0 && <Divider sx={{ my: 1 }} />}
                                <Box>
                                    <Typography variant="caption" color="text.secondary">
                                        {key}
                                    </Typography>
                                    <Typography
                                        variant="body2"
                                        sx={{
                                            fontFamily: "monospace",
                                            whiteSpace: "pre-wrap",
                                            wordBreak: "break-word",
                                        }}
                                    >
                                        {formatValue(value)}
                                    </Typography>
                                </Box>
                            </Box>
                        ))}
                    </Box>
                </Paper>
            )}
        </Box>
    );
}
