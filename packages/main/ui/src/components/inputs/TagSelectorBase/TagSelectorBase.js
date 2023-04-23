import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useQuery } from "@apollo/client";
import { BookmarkFor, TagSortBy } from "@local/consts";
import { Autocomplete, Chip, ListItemText, MenuItem, TextField, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { tagFindMany } from "../../../api/generated/endpoints/tag_findMany";
import { PubSub } from "../../../utils/pubsub";
import { BookmarkButton } from "../../buttons/BookmarkButton/BookmarkButton";
export const TagSelectorBase = ({ disabled, handleTagsUpdate, tags, placeholder, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const handleTagAdd = useCallback((tag) => {
        handleTagsUpdate([...tags, tag]);
    }, [handleTagsUpdate, tags]);
    const handleTagRemove = useCallback((tag) => {
        handleTagsUpdate(tags.filter(t => t.tag !== tag.tag));
    }, [handleTagsUpdate, tags]);
    const [inputValue, setInputValue] = useState("");
    const clearText = useCallback(() => { setInputValue(""); }, []);
    const onChange = useCallback((change) => {
        setInputValue(change.target.value.replace(/[,;]/g, ""));
    }, []);
    const onKeyDown = useCallback((event) => {
        let tagLabel;
        if (event.code === "Comma" || event.code === "Semicolon") {
            tagLabel = inputValue;
        }
        else if (event.code === "enter" && event.target.value) {
            tagLabel = inputValue + event.key;
        }
        else
            return;
        tagLabel = tagLabel.replace(/[,;]/g, "");
        if (tagLabel.length < 2) {
            PubSub.get().publishSnack({ messageKey: "TagTooShort", severity: "Error" });
            return;
        }
        if (tagLabel.length > 30) {
            PubSub.get().publishSnack({ messageKey: "TagTooLong", severity: "Error" });
            return;
        }
        const isSelected = tags.some(t => t.tag === tagLabel);
        if (isSelected) {
            PubSub.get().publishSnack({ messageKey: "TagAlreadySelected", severity: "Error" });
            return;
        }
        handleTagAdd({ tag: tagLabel });
        clearText();
    }, [clearText, handleTagAdd, inputValue, tags]);
    const onInputSelect = useCallback((tag) => {
        setInputValue("");
        const isSelected = tags.some(t => t.tag === tag.tag);
        if (isSelected)
            handleTagRemove(tag);
        else
            handleTagAdd(tag);
    }, [handleTagAdd, handleTagRemove, tags]);
    const onChipDelete = useCallback((tag) => {
        handleTagRemove(tag);
    }, [handleTagRemove]);
    const tagsRef = useRef(null);
    useEffect(() => {
        if (!tagsRef.current)
            return;
        tags.forEach(tag => {
            if (!tagsRef.current[tag.tag])
                tagsRef.current[tag.tag] = tag;
        });
    }, [tags]);
    const { data: autocompleteData, refetch: refetchAutocomplete } = useQuery(tagFindMany, {
        variables: {
            input: {
                excludeIds: tagsRef.current !== null ?
                    Object.values(tagsRef.current)
                        .filter(t => t.id && t.tag.toLowerCase().includes(inputValue.toLowerCase()))
                        .map(t => t.id) :
                    [],
                searchString: inputValue,
                sortBy: TagSortBy.BookmarksDesc,
                take: 25,
            },
        },
    });
    useEffect(() => { refetchAutocomplete(); }, [inputValue, refetchAutocomplete]);
    useEffect(() => {
        if (!autocompleteData)
            return;
        const queried = autocompleteData.tags.edges.map(({ node }) => node);
        queried.forEach(tag => {
            if (!tagsRef.current)
                tagsRef.current = {};
            tagsRef.current[tag.tag] = tag;
        });
    }, [autocompleteData, tagsRef]);
    const autocompleteOptions = useMemo(() => {
        if (!autocompleteData)
            return [];
        const queried = autocompleteData.tags.edges.map(({ node }) => node);
        const known = tagsRef.current ?
            Object.values(tagsRef.current)
                .filter(tag => tag.tag.toLowerCase().includes(inputValue.toLowerCase())) :
            [];
        return [...queried, ...known];
    }, [autocompleteData, inputValue, tagsRef]);
    const handleIsBookmarked = useCallback((tag, isBookmarked) => {
        if (!tagsRef.current)
            tagsRef.current = {};
        tagsRef.current[tag] = { ...tagsRef.current[tag], isBookmarked };
    }, [tagsRef]);
    return (_jsx(Autocomplete, { id: "tags-input", disabled: disabled, fullWidth: true, multiple: true, freeSolo: true, options: autocompleteOptions, getOptionLabel: (o) => (typeof o === "string" ? o : o.tag), inputValue: inputValue, noOptionsText: t("NoSuggestions"), limitTags: 3, onClose: clearText, value: tags, filterOptions: (options, params) => options.filter(o => !tags.some(t => t.tag === o.tag)), renderTags: (value, getTagProps) => value.map((option, index) => (_jsx(Chip, { variant: "filled", label: option.tag, ...getTagProps({ index }), onDelete: () => onChipDelete(option), sx: {
                backgroundColor: palette.mode === "light" ? "#8148b0" : "#8148b0",
                color: "white",
            } }))), renderOption: (props, option) => (_jsxs(MenuItem, { ...props, onClick: () => onInputSelect(option), children: [_jsx(ListItemText, { children: option.tag }), _jsx(BookmarkButton, { objectId: option.id ?? "", bookmarkFor: BookmarkFor.Tag, isBookmarked: option.you.isBookmarked, bookmarks: option.bookmarks, onChange: (isBookmarked) => { handleIsBookmarked(option.tag, isBookmarked); } })] })), renderInput: (params) => (_jsx(TextField, { value: inputValue, onChange: onChange, placeholder: placeholder ?? t("TagSelectorPlaceholder"), InputProps: params.InputProps, inputProps: params.inputProps, onKeyDown: onKeyDown, fullWidth: true, sx: { paddingRight: 0, minWidth: "250px" } })) }));
};
//# sourceMappingURL=TagSelectorBase.js.map