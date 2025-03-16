import { LinearProgress, styled } from "@mui/material";
import { MarkdownDisplay } from "components/text/MarkdownDisplay.js";
import { useMemo } from "react";
import { ContentCollapse } from "../ContentCollapse/ContentCollapse.js";
import { TextCollapseProps } from "../types.js";

const DEFAULT_LOADING_LINES = 1;

const LoadingLine = styled(LinearProgress)(({ theme }) => ({
    borderRadius: theme.spacing(2),
    width: "100%",
    // eslint-disable-next-line no-magic-numbers
    height: theme.spacing(1.5),
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(2),
    opacity: 0.5,
}));

const markdownDisplayStyle = { marginTop: 0 } as const;

export function TextCollapse({
    helpText,
    isOpen,
    loading,
    loadingLines,
    onOpenChange,
    title,
    text,
}: TextCollapseProps) {
    const lines = useMemo(function generateLoadingLines() {
        if (loading !== true) return null;
        return Array.from({ length: loadingLines ?? DEFAULT_LOADING_LINES }, (_, i) => (
            <LoadingLine
                key={`loading-line-${i}`} // Fine to use index as key here
                color="inherit"
            />
        ));
    }, [loading, loadingLines]);

    if ((!text || text.trim().length === 0) && !loading) return null;
    return (
        <ContentCollapse
            helpText={helpText}
            isOpen={isOpen}
            onOpenChange={onOpenChange}
            title={title}
        >
            {text ? <MarkdownDisplay sx={markdownDisplayStyle} content={text} /> : lines}
        </ContentCollapse>
    );
}
