import Box from "@mui/material/Box";
import Snackbar from "@mui/material/Snackbar";
import type { SxProps, Theme } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useDebugStore } from "../../stores/debugStore.js";

// Constants
const COLOR_HUE_MAX = 360;
const HIGHLIGHT_DURATION_MS = 1000;
const CIRCLE_SIZE_PX = 20;
const COPY_FEEDBACK_DURATION_MS = 2000;
const SNACKBAR_AUTO_HIDE_DURATION = 2000;

// Snackbar position
const snackbarPosition = {
    vertical: "bottom" as const,
    horizontal: "center" as const,
};

/**
 * Generate a color from traceId
 */
function getColor(traceId: string): string {
    const hash = traceId.split("").reduce((acc, char) => acc + char.charCodeAt(0), 0);
    return `hsl(${hash % COLOR_HUE_MAX}, 70%, 50%)`;
}

// Box styles
const containerStyles: SxProps<Theme> = {
    position: "fixed",
    bottom: 10,
    right: 10,
    display: "flex",
    flexDirection: "row-reverse",
    flexWrap: "wrap",
    gap: "5px",
    zIndex: 9999,
    maxWidth: "200px",
};

/**
 * Create styles for a debug circle
 */
function getCircleStyles(traceId: string, isHighlighted: boolean, isCopied: boolean): SxProps<Theme> {
    return {
        width: `${CIRCLE_SIZE_PX}px`,
        height: `${CIRCLE_SIZE_PX}px`,
        borderRadius: "50%",
        backgroundColor: getColor(traceId),
        filter: isHighlighted
            ? "brightness(1.5)"
            : isCopied
                ? "brightness(1.2) saturate(1.5)"
                : "brightness(1)",
        transition: "filter 0.3s",
        boxShadow: "0 0 3px rgba(0,0,0,0.3)",
        cursor: "pointer",
        "&:hover": {
            transform: "scale(1.1)",
            boxShadow: "0 0 5px rgba(0,0,0,0.5)",
        },
    };
}

interface TraceCircleProps {
    traceId: string;
    count: number;
    data: any;
    isHighlighted: boolean;
    isCopied: boolean;
    onCopy: (traceId: string, data: any) => void;
}

/**
 * Individual trace circle component to avoid linter issues with inline functions
 */
function TraceCircle({ traceId, count, data, isHighlighted, isCopied, onCopy }: TraceCircleProps): JSX.Element {
    const circleStyle = getCircleStyles(traceId, isHighlighted, isCopied);
    const tooltipText = `Trace: ${traceId}\nCount: ${count}\nData: ${JSON.stringify(data, null, 2)}`;

    const handleClick = useCallback(() => {
        onCopy(traceId, data);
    }, [traceId, data, onCopy]);

    return (
        <Box
            key={traceId}
            sx={circleStyle}
            title={tooltipText}
            onClick={handleClick}
        />
    );
}

/**
 * Debug Component for displaying trace information
 * Only renders in development mode
 */
export function DebugComponent(): JSX.Element | null {
    const traces = useDebugStore((state) => state.traces);
    const [highlighted, setHighlighted] = useState<Record<string, { count: number } | null>>({});
    const [copiedTrace, setCopiedTrace] = useState<string | null>(null);
    const [snackbarOpen, setSnackbarOpen] = useState(false);

    // Highlight circles when new data is added
    useEffect(() => {
        const newHighlights: Record<string, { count: number }> = {};

        Object.keys(traces).forEach((traceId) => {
            const prevCount = highlighted[traceId]?.count || 0;
            if (traces[traceId].count > prevCount) {
                newHighlights[traceId] = { count: traces[traceId].count };

                // Set a timeout to remove the highlight after 1 second
                setTimeout(() => {
                    setHighlighted((prev) => ({ ...prev, [traceId]: null }));
                }, HIGHLIGHT_DURATION_MS);
            }
        });

        if (Object.keys(newHighlights).length > 0) {
            setHighlighted((prev) => ({ ...prev, ...newHighlights }));
        }
    }, [traces, highlighted]);

    // Handle copying trace data to clipboard
    const handleCopyTrace = useCallback(function handleCopyTraceCallback(traceId: string, data: any): void {
        const jsonStr = JSON.stringify(data, null, 2);
        navigator.clipboard.writeText(jsonStr).then(() => {
            setCopiedTrace(traceId);
            setSnackbarOpen(true);

            // Reset copied state after a delay
            setTimeout(() => {
                setCopiedTrace(null);
            }, COPY_FEEDBACK_DURATION_MS);
        }).catch(err => {
            console.error("Failed to copy debug data:", err);
        });
    }, []);

    const handleCloseSnackbar = useCallback(function handleCloseSnackbarCallback(): void {
        setSnackbarOpen(false);
    }, []);

    // Only render in development mode
    if (process.env.NODE_ENV !== "development") {
        return null;
    }

    return (
        <>
            <Box sx={containerStyles}>
                {Object.entries(traces).map(([traceId, { count, data }]) => (
                    <TraceCircle
                        key={traceId}
                        traceId={traceId}
                        count={count}
                        data={data}
                        isHighlighted={Boolean(highlighted[traceId])}
                        isCopied={traceId === copiedTrace}
                        onCopy={handleCopyTrace}
                    />
                ))}
            </Box>

            <Snackbar
                open={snackbarOpen}
                autoHideDuration={SNACKBAR_AUTO_HIDE_DURATION}
                onClose={handleCloseSnackbar}
                message={copiedTrace ? `Copied data from ${copiedTrace}` : "Copied debug data"}
                anchorOrigin={snackbarPosition}
            />
        </>
    );
} 
