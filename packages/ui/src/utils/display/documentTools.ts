/**
 * Removes all highlight spans from the given element, and combines the text nodes within.
 * @param highlightClass The class name of the highlight spans to remove.
 * @param element The element to remove the highlight spans from. If not given, the entire document is used.
 */
export const removeHighlights = (highlightClass: string, element?: HTMLElement) => {
    const root = element || document.body; //document.getElementById('content-wrap');
    if (!root) return;
    const highlightedElements = root.querySelectorAll(`.${highlightClass}`);
    highlightedElements.forEach((root) => {
        const parent = root.parentNode;
        if (!parent) return;
        const text = root.textContent;
        if (!text) return;
        const textNode = document.createTextNode(text);
        parent.replaceChild(textNode, root);
    });
};

/**
 * Finds all text nodes in the given element that are not hidden by another element on top of them.
 * @param element The element to find the text nodes in. If not given, the entire document is used.
 * @returns An array of text nodes.
 */
export const getTextNodes = (element?: HTMLElement) => {
    const textNodes: Text[] = [];
    const root = element || document.body; //document.getElementById('content-wrap');
    if (!root) return textNodes;
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, null);
    let node = walker.nextNode();
    while (node) {
        // Check if text is hidden by another element
        const hasOffsetParent = node.parentElement?.offsetParent !== null;
        if (hasOffsetParent) {
            textNodes.push(node as Text);
        }
        node = walker.nextNode();
    }
    return textNodes;
};

/**
 * Normalizes the given text by replacing diacritics with their base characters.
 * Keeps whitespace, punctuation, numbers, and emojis intact.
 * @param text The text to normalize.
 * @returns The normalized text.
 */
export const normalizeText = (text: string) => {
    return text.normalize("NFD").replace(/[\u0300-\u036f]/g, "");
};

/**
 * Removes emojis from the given text.
 * @param text The text to remove emojis from.
 * @returns The text without emojis.
 */
export const removeEmojis = (text: string) => {
    return text.replace(/([\u2700-\u27BF]|[\uE000-\uF8FF]|\uD83C[\uDC00-\uDFFF]|\uD83D[\uDC00-\uDFFF]|[\u2011-\u26FF]|\uD83E[\uDD10-\uDDFF])/g, "");
};

/**
 * Removes punctuation from the given text.
 * @param text The text to remove punctuation from.
 * @returns The text without punctuation.
 */
export const removePunctuation = (text: string) => {
    return text.replace(/[.,/#!$%^&*;:{}=\-_`~()]/g, "");
};

/**
 * Wraps matches in the given text node with a span element.
 * @param textNode The text node to highlight matches in.
 * @param searchRegex The regex used to search for matches.
 * @param wrapperClass The class name of the wrapper span, which is wrapped around the entire text node.
 * @param spanClass The class name of the span, which is wrapped around each match.
 * @returns The highlighted span elements
 */
export const wrapMatches = (textNode: Text, searchRegex: RegExp, wrapperClass: string, spanClass: string): HTMLSpanElement[] => {
    const highlightSpans: HTMLSpanElement[] = [];
    if (!textNode.textContent) return highlightSpans;
    // Normalize the text node
    const normalizedText = normalizeText(textNode.textContent);
    // Find all matching sections within the text node
    const matches = normalizedText.match(searchRegex);
    if (!matches) return highlightSpans;
    // Loop through all matches, to build an element that will replace the text node. 
    // This element will have the same text as the text node, but with the matching sections each 
    // wrapped in a span with a custom class
    const newElement = document.createElement("span");
    newElement.className = wrapperClass;
    let lastIndex = 0;
    matches.forEach((match) => {
        // Get the index of the match
        const matchIndex = normalizedText.indexOf(match, lastIndex);
        // Get the text before the match
        const textBeforeMatch = (textNode.textContent ?? "").slice(lastIndex, matchIndex);
        // Create a text node for the text before the match
        const textBeforeMatchNode = document.createTextNode(textBeforeMatch);
        // Append the text node to the new element
        newElement.appendChild(textBeforeMatchNode);
        // Create a span for the match
        const matchSpan = document.createElement("span");
        // Add the custom class to the span
        matchSpan.classList.add(spanClass);
        // Create a text node for the match
        const matchTextNode = document.createTextNode(match);
        // Append the text node to the span
        matchSpan.appendChild(matchTextNode);
        // Append the span to the new element
        newElement.appendChild(matchSpan);
        // Add the span to the list of highlight spans
        highlightSpans.push(matchSpan);
        // Update the last index
        lastIndex = matchIndex + match.length;
    });
    // Get the text after the last match
    const textAfterLastMatch = (textNode.textContent ?? "").slice(lastIndex);
    // Create a text node for the text after the last match
    const textAfterLastMatchNode = document.createTextNode(textAfterLastMatch);
    // Append the text node to the new element
    newElement.appendChild(textAfterLastMatchNode);
    // Replace the text node with the new element
    textNode.replaceWith(newElement);
    return highlightSpans;
};
