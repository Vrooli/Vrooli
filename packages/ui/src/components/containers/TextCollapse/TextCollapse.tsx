import { LinearProgress } from '@mui/material';
import Markdown from 'markdown-to-jsx';
import { useMemo } from 'react';
import { ContentCollapse } from '../ContentCollapse/ContentCollapse';
import { TextCollapseProps } from '../types';

export function TextCollapse({
    helpText,
    isOpen,
    loading,
    loadingLines,
    onOpenChange,
    session,
    title,
    text,
}: TextCollapseProps) {

    const lines = useMemo(() => {
        if (!loading) return null;
        return Array.from({ length: loadingLines ?? 1 }, (_, i) => (
            <LinearProgress color="inherit" sx={{
                borderRadius: 2,
                width: '100%',
                height: 12,
                marginTop: 1,
                marginBottom: 2,
                opacity: 0.5,
            }} />
        ))
    }, [loading, loadingLines])

    if ((!text || text.trim().length === 0) && !loading) return null;
    return (
        <ContentCollapse
            helpText={helpText}
            isOpen={isOpen}
            onOpenChange={onOpenChange}
            title={title}
        >
            {text ? <Markdown style={{ marginTop: 0 }}>{text}</Markdown> : lines}
        </ContentCollapse>
    )
}