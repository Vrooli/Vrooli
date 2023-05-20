import { useQuery } from "@apollo/client";
import { BookmarkFor, Tag, TagSearchInput, TagSearchResult, TagSortBy } from "@local/shared";
import { Autocomplete, Chip, ListItemText, MenuItem, TextField, useTheme } from "@mui/material";
import { tagFindMany } from "api/generated/endpoints/tag_findMany";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Wrap } from "types";
import { useDebounce } from "utils/hooks/useDebounce";
import { PubSub } from "utils/pubsub";
import { TagShape } from "utils/shape/models/tag";
import { TagSelectorBaseProps } from "../types";

export const TagSelectorBase = ({
    disabled,
    handleTagsUpdate,
    tags,
    placeholder,
    zIndex,
}: TagSelectorBaseProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const handleTagAdd = useCallback((tag: TagShape) => {
        handleTagsUpdate([...tags, tag]);
    }, [handleTagsUpdate, tags]);
    const handleTagRemove = useCallback((tag: TagShape) => {
        handleTagsUpdate(tags.filter(t => t.tag !== tag.tag));
    }, [handleTagsUpdate, tags]);

    const [inputValue, setInputValue] = useState<string>("");
    const clearText = useCallback(() => { setInputValue(""); }, []);
    const onChange = useCallback((change: any) => {
        // Remove invalid characters (i.e. ',' or ';')
        setInputValue(change.target.value.replace(/[,;]/g, ""));
    }, []);
    const onKeyDown = useCallback((event: any) => {
        let tagLabel;
        // Check if the user pressed ',' or ';'
        if (event.code === "Comma" || event.code === "Semicolon") {
            tagLabel = inputValue;
        }
        // Check if the user pressed enter
        else if (event.code === "enter" && event.target.value) {
            tagLabel = inputValue + event.key;
        }
        else return;
        // Remove invalid characters (i.e. ',' or ';')
        tagLabel = tagLabel.replace(/[,;]/g, "");
        // Check if tag is valid length
        if (tagLabel.length < 2) {
            PubSub.get().publishSnack({ messageKey: "TagTooShort", severity: "Error" });
            return;
        }
        if (tagLabel.length > 30) {
            PubSub.get().publishSnack({ messageKey: "TagTooLong", severity: "Error" });
            return;
        }
        // Determine if tag is already selected
        const isSelected = tags.some(t => t.tag === tagLabel);
        if (isSelected) {
            PubSub.get().publishSnack({ messageKey: "TagAlreadySelected", severity: "Error" });
            return;
        }
        // Add tag
        handleTagAdd({ tag: tagLabel });
        // Clear input
        clearText();
    }, [clearText, handleTagAdd, inputValue, tags]);

    const onInputSelect = useCallback((tag: Tag) => {
        setInputValue("");
        // Determine if tag is already selected
        const isSelected = tags.some(t => t.tag === tag.tag);
        if (isSelected) handleTagRemove(tag);
        else handleTagAdd(tag);
    }, [handleTagAdd, handleTagRemove, tags]);
    const onChipDelete = useCallback((tag: TagShape | Tag) => {
        handleTagRemove(tag);
    }, [handleTagRemove]);

    // Map of tag strings to queried tag data, so we can exclude tags that have already been queried before
    type TagsRef = { [key: string]: TagShape | Tag }
    const tagsRef = useRef<TagsRef | null>(null);
    // Whenever selected tags change, add unknown tags to the tag map
    useEffect(() => {
        if (!tagsRef.current) return;
        tags.forEach(tag => {
            if (!(tagsRef.current as TagsRef)[tag.tag]) (tagsRef.current as TagsRef)[tag.tag] = tag;
        });
    }, [tags]);

    const { data: autocompleteData, refetch: refetchAutocomplete } = useQuery<Wrap<TagSearchResult, "tags">, Wrap<TagSearchInput, "input">>(tagFindMany, {
        variables: {
            input: {
                // Exclude tags that have already been fully queried, and match the search string
                // (i.e. in tag map, and have an ID)
                excludeIds: tagsRef.current !== null ?
                    Object.values(tagsRef.current)
                        .filter(t => (t as Tag).id && t.tag.toLowerCase().includes(inputValue.toLowerCase()))
                        .map(t => (t as Tag).id) as string[] :
                    [],
                searchString: inputValue,
                sortBy: TagSortBy.Top,
                take: 25,
            },
        },
    });
    const debounceRefetch = useDebounce(() => refetchAutocomplete(), 250);
    useEffect(() => { debounceRefetch({}); }, [debounceRefetch, inputValue]);

    /**
     * Store queried tags in the tag ref
     */
    useEffect(() => {
        if (!autocompleteData) return;
        const queried = autocompleteData.tags.edges.map(({ node }) => node);
        queried.forEach(tag => {
            if (!tagsRef.current) tagsRef.current = {};
            (tagsRef.current as TagsRef)[tag.tag] = tag;
        });
    }, [autocompleteData, tagsRef]);

    const autocompleteOptions: (TagShape | Tag)[] = useMemo(() => {
        if (!autocompleteData) return [];
        // Find queried
        const queried = autocompleteData.tags.edges.map(({ node }) => node);
        // Find already known, that match the search string
        const known = tagsRef.current ?
            Object.values(tagsRef.current)
                .filter(tag => tag.tag.toLowerCase().includes(inputValue.toLowerCase())) :
            [];
        // Return all queried and known
        return [...queried, ...known];
    }, [autocompleteData, inputValue, tagsRef]);

    const handleIsBookmarked = useCallback((tag: string, isBookmarked: boolean) => {
        // Update tag ref
        if (!tagsRef.current) tagsRef.current = {};
        ((tagsRef.current as TagsRef)[tag] as any) = { ...(tagsRef.current as TagsRef)[tag], isBookmarked };
    }, [tagsRef]);

    return (
        <Autocomplete
            id="tags-input"
            disabled={disabled}
            fullWidth
            multiple
            freeSolo={true}
            options={autocompleteOptions}
            getOptionLabel={(o: string | TagShape | Tag) => (typeof o === "string" ? o : o.tag)}
            inputValue={inputValue}
            noOptionsText={t("NoSuggestions")}
            limitTags={3}
            onClose={clearText}
            value={tags}
            // Filter out what has already been selected
            filterOptions={(options, params) => options.filter(o => !tags.some(t => t.tag === o.tag))}
            renderTags={(value, getTagProps) =>
                value.map((option: TagShape | Tag, index) => (
                    <Chip
                        variant="filled"
                        label={option.tag}
                        {...getTagProps({ index })}
                        onDelete={() => onChipDelete(option)}
                        sx={{
                            backgroundColor: palette.mode === "light" ? "#8148b0" : "#8148b0", //'#a068ce',
                            color: "white",
                        }}
                    />
                ),
                )}
            renderOption={(props, option: TagShape | Tag) => (
                <MenuItem
                    {...props}
                    onClick={() => onInputSelect(option as Tag)} //TODO
                >
                    <ListItemText>{option.tag}</ListItemText>
                    <BookmarkButton
                        objectId={(option as Tag).id ?? ""}
                        bookmarkFor={BookmarkFor.Tag}
                        isBookmarked={(option as Tag).you.isBookmarked}
                        bookmarks={(option as Tag).bookmarks}
                        onChange={(isBookmarked) => { handleIsBookmarked(option.tag, isBookmarked); }}
                        zIndex={zIndex}
                    />
                </MenuItem>
            )}
            renderInput={(params) => (
                <TextField
                    value={inputValue}
                    onChange={onChange}
                    placeholder={placeholder ?? t("TagSelectorPlaceholder")}
                    InputProps={params.InputProps}
                    inputProps={params.inputProps}
                    onKeyDown={onKeyDown}
                    fullWidth
                    sx={{ paddingRight: 0, minWidth: "250px" }}
                />
            )}
        />
    );
};
