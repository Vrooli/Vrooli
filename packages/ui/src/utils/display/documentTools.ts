// Names of all highlight classes. Must be defined in 
// theme styles in App.tsx
/** Style for current search match */
export const SEARCH_HIGHLIGHT_CURRENT = "search-highlight-current";
/** Style for element search map is placed in */
export const SEARCH_HIGHLIGHT_ELEMENT = "search-highlight-wrap";
/** Style for wrapper element that the search element is placed in */
export const SEARCH_HIGHLIGHT_WRAPPER = "search-highlight";
/** Style for tutorial element highlight */
export const TUTORIAL_HIGHLIGHT = "tutorial-highlight";
/** Style for snack message highlight */
export const SNACK_HIGHLIGHT = "snack-highlight";
/** Style for node with error */
export const NODE_HIGHLIGHT_ERROR = "node-highlight-error";
/** Style for node with warning */
export const NODE_HIGHLIGHT_WARNING = "node-highlight-warning";
/** Style for selected node */
export const NODE_HIGHLIGHT_SELECTED = "node-highlight-selected";

/**
 * Highlights all instances of the search term. Accomplishes this by doing the following: 
 * 1. Remove all previous highlights
 * 2. Finds all text nodes in the document
 * 3. Maps diacritics from the search term and text nodes to their base characters
 * 4. Checks if the search term is a substring of the text node, while adhering to the search options (case sensitive, whole word, regex)
 * 5. Highlights the text node if it matches the search term, by wrapping it in a span with a custom class (custom class is necessary to remove the highlight later)
 * @param searchString The search term
 * @param isCaseSensitive Whether or not the search should be case sensitive
 * @param isWholeWord Whether or not the search should be whole word
 * @param isRegex Whether or not the search should be regex
 * @returns Highlight spans
 */
export function highlightText(
    searchString: string,
    isCaseSensitive: boolean,
    isWholeWord: boolean,
    isRegex: boolean,
): HTMLSpanElement[] {
    // Remove all previous highlights
    removeHighlights(SEARCH_HIGHLIGHT_WRAPPER); // General highlight class
    removeHighlights(SEARCH_HIGHLIGHT_CURRENT); // Highlight class for the current match
    removeHighlights(SEARCH_HIGHLIGHT_ELEMENT); // Wrapper class for a highlight's text node
    // If text is empty, return
    if (searchString.trim().length === 0) return [];
    // Finds all text nodes in the document
    const textNodes: Text[] = getTextNodes();
    // Normalize the search term
    const normalizedSearchString = normalizeText(searchString);
    // Build the regex
    let regexString = normalizedSearchString;
    // If not regex, escape regex characters
    if (!isRegex) { regexString = regexString.replace(/[-[\]{}()*+?.,\\^$|#\s]/g, "\\$&"); }
    // If whole word, wrap the search term in word boundaries
    if (isWholeWord) { regexString = `\\b${regexString}\\b`; }
    // Create global regex expression 
    const regex = new RegExp(regexString, isCaseSensitive ? "g" : "gi");
    // Loop through all text nodes, and store highlights. 
    // These will be used for previous/next buttons
    const highlightSpans: HTMLSpanElement[] = [];
    textNodes.forEach((textNode) => {
        const spans = wrapMatches(textNode, regex);
        highlightSpans.push(...spans);
    });
    // If there is at least one highlight, change the first highlight's class to 'search-highlight-current'
    if (highlightSpans.length > 0) {
        highlightSpans[0].classList.add(SEARCH_HIGHLIGHT_CURRENT);
    }
    return highlightSpans;
}

/**
 * Adds a highlight class to the provided element.
 * @param highlightClass The class name of the highlight spans or elements to add.
 * @param element The element to add the highlight class to.
 */
export function addHighlight(highlightClass: string, element: Element) {
    element.classList.add(highlightClass);
}

/**
 * Removes all highlight spans from the given element and removes highlight classes from non-text elements.
 * @param highlightClass The class name of the highlight spans or elements to remove.
 * @param element The element to remove the highlight spans from. If not given, the entire document is used.
 */
export function removeHighlights(highlightClass: string, element?: Element) {
    const root = element || document.body;
    if (!root) return;
    let highlightedElements: Element[] = [];
    // Check children of the root element for highlight spans
    const highlightedChildren = root.querySelectorAll(`.${highlightClass}`);
    highlightedElements = Array.from(highlightedChildren);
    // Check the root element itself for highlight spans
    if (root.classList.contains(highlightClass)) {
        highlightedElements.push(root);
    }
    highlightedElements.forEach((element) => {
        if (element instanceof HTMLSpanElement && element.classList.contains(SEARCH_HIGHLIGHT_ELEMENT)) {
            const parent = element.parentNode;
            if (!parent) return;
            const text = element.textContent;
            if (!text) return;
            const textNode = document.createTextNode(text);
            parent.replaceChild(textNode, element);
        } else {
            // Remove the highlight class from non-text elements
            element.classList.remove(highlightClass);
        }
    });
}

/**
 * Finds all text nodes in the given element that are not hidden by another element on top of them.
 * @param element The element to find the text nodes in. If not given, the entire document is used.
 * @returns An array of text nodes.
 */
export function getTextNodes(element?: HTMLElement) {
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
}

/**
 * Normalizes the given text by replacing diacritics with their base characters.
 * Keeps whitespace, punctuation, numbers, and emojis intact.
 * @param text The text to normalize.
 * @returns The normalized text.
 */
export function normalizeText(text: string) {
    return text.normalize("NFD").replace(/[\u0300-\u036f]/g, "");
}

/**
 * Removes emojis from the given text.
 * @param text The text to remove emojis from.
 * @returns The text without emojis.
 */
export function removeEmojis(text: string) {
    return text.replace(/([\u2700-\u27BF]|[\uE000-\uF8FF]|\uD83C[\uDC00-\uDFFF]|\uD83D[\uDC00-\uDFFF]|[\u2011-\u26FF]|\uD83E[\uDD10-\uDDFF])/g, "");
}

/**
 * Removes punctuation from the given text.
 * @param text The text to remove punctuation from.
 * @returns The text without punctuation.
 */
export function removePunctuation(text: string) {
    return text.replace(/[.,/#!$%^&*;:{}=\-_`~()]/g, "");
}

/**
 * Wraps matches in the given text node with a span element.
 * @param textNode The text node to highlight matches in.
 * @param searchRegex The regex used to search for matches.
 * @returns The highlighted span elements
 */
export function wrapMatches(
    textNode: Text,
    searchRegex: RegExp,
): HTMLSpanElement[] {
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
    newElement.className = SEARCH_HIGHLIGHT_ELEMENT;
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
        matchSpan.classList.add(SEARCH_HIGHLIGHT_WRAPPER);
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
}
