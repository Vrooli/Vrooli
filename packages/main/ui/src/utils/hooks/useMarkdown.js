import { useEffect, useState } from "react";
export const useMarkdown = (importedFile, shapingFunction) => {
    const [markdown, setMarkdown] = useState("");
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
};
//# sourceMappingURL=useMarkdown.js.map