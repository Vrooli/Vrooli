import { TextCollapseProps } from '../types';
import { ContentCollapse } from 'components';
import Markdown from 'markdown-to-jsx';

export function TextCollapse({
    helpText,
    isOpen,
    onOpenChange,
    showOnNoText,
    title,
    text,
}: TextCollapseProps) {
    if ((!text || text.trim().length === 0) && !showOnNoText) return null;
    return (
        <ContentCollapse
            helpText={helpText}
            isOpen={isOpen}
            onOpenChange={onOpenChange}
            title={title}
        >
            {text ? <Markdown>{text}</Markdown> : null}
        </ContentCollapse>
    )
}