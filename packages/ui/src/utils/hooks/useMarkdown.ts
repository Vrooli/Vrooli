import { useEffect, useState } from 'react';

export const useMarkdown = (importedFile: string, shapingFunction?: (markdown: string) => string): string => {
    const [markdown, setMarkdown] = useState<string>('');

    useEffect(() => {
        fetch(importedFile)
            .then(response => response.text())
            .then(text => {
                const shapedMarkdown = shapingFunction ? shapingFunction(text) : text;
                setMarkdown(shapedMarkdown);
            })
            .catch(error => {
                console.error(`Failed to fetch markdown from ${importedFile}: ${error}`);
            });
    }, [importedFile, shapingFunction]);

    return markdown;
}