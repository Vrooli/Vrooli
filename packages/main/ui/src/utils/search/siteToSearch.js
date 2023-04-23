import i18next from "i18next";
import { getSiteLanguage } from "../authentication/session";
import { normalizeText, removeEmojis, removePunctuation } from "../display/documentTools";
export const shapeSearchText = (text) => {
    if (!text) {
        console.warn("No text provided to shapeSearchText");
        return "";
    }
    let shaped = text.trim();
    shaped = normalizeText(shaped);
    shaped = removePunctuation(shaped);
    shaped = removeEmojis(shaped);
    shaped = shaped.toLowerCase();
    return shaped;
};
export const translateSearchItems = (items, session) => {
    const lng = getSiteLanguage(session);
    return items.map(item => {
        const label = i18next.t(`common:${item.label}`, { ...(item.labelArgs ?? {}), lng });
        const keywords = [shapeSearchText(label)];
        const unshapedKeywords = [label];
        for (const keyword of item.keywords ?? []) {
            if (typeof keyword === "string") {
                const keywordText = i18next.t(keyword);
                keywords.push(shapeSearchText(keywordText));
                unshapedKeywords.push(keywordText);
            }
            else {
                const keywordText = i18next.t(keyword.key, { ...keyword, lng });
                keywords.push(shapeSearchText(keywordText));
                unshapedKeywords.push(keywordText);
            }
        }
        return {
            label,
            keywords,
            unshapedKeywords,
            value: item.value,
        };
    });
};
export const findSearchResults = (items, { inputValue }) => {
    console.log("findSearchResults start", { ...items }, inputValue);
    const shapedTerm = shapeSearchText(inputValue);
    const matches = items.filter(item => item.keywords?.some(keyword => keyword.includes(shapedTerm)));
    return matches.sort((a, b) => {
        const aExact = a.keywords?.some(keyword => keyword === shapedTerm) ?? false;
        const bExact = b.keywords?.some(keyword => keyword === shapedTerm) ?? false;
        if (aExact && !bExact) {
            return -1;
        }
        else if (!aExact && bExact) {
            return 1;
        }
        else {
            const aMatchCount = a.keywords?.filter(keyword => keyword.includes(shapedTerm)).length ?? 0;
            const bMatchCount = b.keywords?.filter(keyword => keyword.includes(shapedTerm)).length ?? 0;
            if (aMatchCount > bMatchCount) {
                return -1;
            }
            else if (aMatchCount < bMatchCount) {
                return 1;
            }
            else {
                return a.label.localeCompare(b.label);
            }
        }
    });
};
//# sourceMappingURL=siteToSearch.js.map