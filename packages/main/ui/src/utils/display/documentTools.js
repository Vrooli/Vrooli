export const removeHighlights = (highlightClass, element) => {
    const root = element || document.body;
    if (!root)
        return;
    const highlightedElements = root.querySelectorAll(`.${highlightClass}`);
    highlightedElements.forEach((root) => {
        const parent = root.parentNode;
        if (!parent)
            return;
        const text = root.textContent;
        if (!text)
            return;
        const textNode = document.createTextNode(text);
        parent.replaceChild(textNode, root);
    });
};
export const getTextNodes = (element) => {
    const textNodes = [];
    const root = element || document.body;
    if (!root)
        return textNodes;
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, null);
    let node = walker.nextNode();
    while (node) {
        const hasOffsetParent = node.parentElement?.offsetParent !== null;
        if (hasOffsetParent) {
            textNodes.push(node);
        }
        node = walker.nextNode();
    }
    return textNodes;
};
export const normalizeText = (text) => {
    return text.normalize("NFD").replace(/[\u0300-\u036f]/g, "");
};
export const removeEmojis = (text) => {
    return text.replace(/([\u2700-\u27BF]|[\uE000-\uF8FF]|\uD83C[\uDC00-\uDFFF]|\uD83D[\uDC00-\uDFFF]|[\u2011-\u26FF]|\uD83E[\uDD10-\uDDFF])/g, "");
};
export const removePunctuation = (text) => {
    return text.replace(/[.,/#!$%^&*;:{}=\-_`~()]/g, "");
};
export const wrapMatches = (textNode, searchRegex, wrapperClass, spanClass) => {
    const highlightSpans = [];
    if (!textNode.textContent)
        return highlightSpans;
    const normalizedText = normalizeText(textNode.textContent);
    const matches = normalizedText.match(searchRegex);
    if (!matches)
        return highlightSpans;
    const newElement = document.createElement("span");
    newElement.className = wrapperClass;
    let lastIndex = 0;
    matches.forEach((match) => {
        const matchIndex = normalizedText.indexOf(match, lastIndex);
        const textBeforeMatch = (textNode.textContent ?? "").slice(lastIndex, matchIndex);
        const textBeforeMatchNode = document.createTextNode(textBeforeMatch);
        newElement.appendChild(textBeforeMatchNode);
        const matchSpan = document.createElement("span");
        matchSpan.classList.add(spanClass);
        const matchTextNode = document.createTextNode(match);
        matchSpan.appendChild(matchTextNode);
        newElement.appendChild(matchSpan);
        highlightSpans.push(matchSpan);
        lastIndex = matchIndex + match.length;
    });
    const textAfterLastMatch = (textNode.textContent ?? "").slice(lastIndex);
    const textAfterLastMatchNode = document.createTextNode(textAfterLastMatch);
    newElement.appendChild(textAfterLastMatchNode);
    textNode.replaceWith(newElement);
    return highlightSpans;
};
//# sourceMappingURL=documentTools.js.map