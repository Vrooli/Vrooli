import { BookmarkFor, DUMMY_ID, endpointGetTags, exists, Tag, TagSearchInput, TagSearchResult, TagSortBy } from "@local/shared";
import { Autocomplete, Chip, CircularProgress, InputAdornment, ListItemText, MenuItem, Popper, PopperProps, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { useField } from "formik";
import { useFetch } from "hooks/useFetch";
import { TagIcon } from "icons";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { PubSub } from "utils/pubsub";
import { TagShape } from "utils/shape/models/tag";
import { TextInput } from "../TextInput/TextInput";
import { TagSelectorBaseProps, TagSelectorProps } from "../types";

/** Removes invalid characters from tag string */
const withoutInvalidChars = (str: string) => str.replace(/[,;]/g, "");

/** Custom Popper component to add scroll handling */
const PopperComponent = ({
    onScrollBottom,
    ...props
}: PopperProps & { onScrollBottom: () => unknown }) => {
    const popperRef = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        const handleScroll = (event) => {
            const target = event.target;
            // Check if we've scrolled to the bottom
            if (target.scrollTop + target.clientHeight >= target.scrollHeight - 10) {
                // Trigger load more function
                onScrollBottom();
            }
        };

        // The grandchild of the Popper is the scrollable element
        const popperNode = popperRef.current;
        const scrollableNode = popperNode?.firstChild?.firstChild;
        if (scrollableNode) {
            scrollableNode.addEventListener("scroll", handleScroll);
        }

        return () => {
            if (scrollableNode) {
                scrollableNode.removeEventListener("scroll", handleScroll);
            }
        };
    }, [onScrollBottom, props.open]); // Re-run effect when open state changes

    return <Popper {...props} ref={popperRef} />;
};

const PAGE_SIZE = 25;

export const TagSelectorBase = ({
    disabled,
    handleTagsUpdate,
    isOptional = true,
    tags,
    placeholder,
    sx,
}: TagSelectorBaseProps) => {
    console.log("tagselectorbase tags", tags);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [inputValue, setInputValue] = useState<string>("");
    // Switch between cursor and offset pagination depending on if there's a search string. 
    // This is because we're using text embedding search to match the search string to tags, 
    // which requires offset pagination.
    const [after, setAfter] = useState<string | null>(null);
    const [offset, setOffset] = useState<number>(0);

    const handleTagAdd = useCallback((tag: string | TagShape) => {
        if (tags.some(t => typeof tag === "string"
            ? (t as unknown as string) === tag
            : (t as TagShape).tag === tag.tag)) return;
        handleTagsUpdate([...tags, typeof tag === "string" ? { __typename: "Tag", id: DUMMY_ID, tag } : tag]);
    }, [handleTagsUpdate, tags]);
    const handleTagRemove = useCallback((tag: string | TagShape) => {
        handleTagsUpdate(tags.filter(t => typeof tag === "string"
            ? (t as unknown as string) !== tag
            : (t as TagShape).tag !== tag.tag));
    }, [handleTagsUpdate, tags]);

    const clearText = useCallback(() => { setInputValue(""); }, []);

    const onChange = useCallback((change: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        const sanitized = withoutInvalidChars(change.target.value);
        console.log("onchange", change.target.value, sanitized);
        setInputValue(sanitized);
    }, []);

    // Detect when the tag should be submitted
    const onKeyDown = useCallback((event: React.KeyboardEvent<HTMLDivElement>) => {
        // Check if the user pressed ',', ';', or enter
        if (!["Comma", "Semicolon", "Enter"].includes(event.code)) return;
        // Remove invalid characters
        const tagLabel = withoutInvalidChars(inputValue);
        // Check if tag is valid length
        if (tagLabel.length < 2) {
            PubSub.get().publish("snack", { messageKey: "TagTooShort", severity: "Error" });
            return;
        }
        if (tagLabel.length > 30) {
            PubSub.get().publish("snack", { messageKey: "TagTooLong", severity: "Error" });
            return;
        }
        // Determine if tag is already selected
        const isSelected = tags.some(t => t.tag === tagLabel);
        if (isSelected) {
            PubSub.get().publish("snack", { messageKey: "TagAlreadySelected", severity: "Error" });
            return;
        }
        // Add tag
        handleTagAdd({ __typename: "Tag", id: DUMMY_ID, tag: tagLabel });
        // Clear input
        clearText();
    }, [clearText, handleTagAdd, inputValue, tags]);

    const onInputSelect = useCallback((tag: TagShape | Tag) => {
        console.log("onInputSelect", tag);
        setInputValue("");
        // Remove tag is already selected
        if (tags.some(t => t.tag === tag.tag)) handleTagRemove(tag);
        // Add otherwise
        else handleTagAdd(tag);
    }, [handleTagAdd, handleTagRemove, tags]);

    const onChipDelete = useCallback((tag: TagShape | Tag) => {
        handleTagRemove(tag);
    }, [handleTagRemove]);

    // Map of tag strings to queried tag data, so we can persist bookmarks
    const tagsRef = useRef<{ [key: string]: TagShape | Tag }>({});

    const { data: autocompleteData, loading } = useFetch<TagSearchInput, TagSearchResult>({
        ...endpointGetTags,
        debounceMs: 250,
        inputs: {
            sortBy: TagSortBy.EmbedTopDesc,
            take: PAGE_SIZE,
            ...(
                inputValue.trim().length > 0
                    ? { offset, searchString: inputValue }
                    : { after }
            ),
        },
    }, [inputValue]);

    useMemo(() => {
        if (inputValue.trim().length > 0) {
            setOffset(0);  // Reset offset when a new search is started
        } else {
            setAfter(null); // Reset cursor when search is cleared
        }
    }, [inputValue]);

    const loadMoreTags = useCallback(() => {
        if (autocompleteData?.pageInfo?.endCursor && autocompleteData.pageInfo.hasNextPage && !inputValue) {
            setAfter(autocompleteData.pageInfo.endCursor);
        } else if (autocompleteData?.pageInfo?.hasNextPage && inputValue) {
            setOffset(prevOffset => prevOffset + PAGE_SIZE);
        }
    }, [autocompleteData, inputValue]);

    const autocompleteOptions: (TagShape | Tag)[] = useMemo(() => {
        if (!autocompleteData) return [];
        // Find queried tags
        const queried = autocompleteData.edges.map(({ node }) => node);
        // Store queried tags in the tag ref
        queried.forEach(tag => {
            if (!tagsRef.current[tag.tag]) tagsRef.current[tag.tag] = tag;
        });
        // Grab tags from the tag ref, in the same order as the queried tags
        const stored = autocompleteData.edges.map(({ node }) => tagsRef.current[node.tag] ?? node);
        // Return stored tags
        return stored;
    }, [autocompleteData, tagsRef]);
    console.log("tag autocomplete options", autocompleteOptions);

    const handleIsBookmarked = useCallback((tag: string, isBookmarked: boolean) => {
        if (!tagsRef.current[tag]) return;
        (tagsRef.current[tag] as TagShape).you = {
            ...((tagsRef.current[tag] as TagShape).you ?? {}),
            __typename: "TagYou" as const,
            isBookmarked,
        } as Tag["you"];
    }, [tagsRef]);

    return (
        <Autocomplete
            id="tags-input"
            disabled={disabled}
            disablePortal
            fullWidth
            multiple
            // Allow all options through the filter - we perform custom filtering
            filterOptions={(options) => options}
            freeSolo={true}
            isOptionEqualToValue={(option, value) => {
                const optionTag = typeof option === "string" ? option : option.tag;
                const valueTag = typeof value === "string" ? value : value.tag;
                return optionTag === valueTag;
            }}
            options={autocompleteOptions}
            getOptionLabel={(o: string | TagShape | Tag) => (typeof o === "string" ? o : o.tag)}
            inputValue={inputValue}
            noOptionsText={t("NoSuggestions")}
            limitTags={3}
            loading={loading}
            value={tags}
            defaultValue={tags}
            PopperComponent={(props) => <PopperComponent {...props} onScrollBottom={loadMoreTags} />}
            renderTags={(value, getTagProps) =>
                value.map((option, index) => {
                    console.log("rendering tag", option, index);
                    return (
                        <Chip
                            {...getTagProps({ index })}
                            id={`tag-chip-${index}`}
                            key={typeof option === "string" ? option : option.tag}
                            variant="filled"
                            label={typeof option === "string" ? option : option.tag}
                            onDelete={() => onChipDelete(option as TagShape | Tag)}
                            sx={{
                                backgroundColor: palette.mode === "light" ? "#8148b0" : "#8148b0", //'#a068ce',
                                color: "white",
                            }}
                        />
                    );
                })}
            renderOption={(props, option) => (
                <MenuItem
                    {...props}
                    onClick={() => {
                        if (typeof option === "string") {
                            const found = autocompleteOptions.find(t => t.tag === option);
                            if (found) onInputSelect(found);
                        } else {
                            onInputSelect(option);
                        }
                    }}
                    selected={tags.some(t => typeof option === "string" ? (t as unknown as string) === option : (t as TagShape | Tag).tag === option.tag)}
                >
                    <ListItemText>{typeof option === "string" ? option : option.tag}</ListItemText>
                    <BookmarkButton
                        objectId={(option as Tag).id ?? ""}
                        bookmarkFor={BookmarkFor.Tag}
                        isBookmarked={(option as Tag).you.isBookmarked}
                        bookmarks={(option as Tag).bookmarks}
                        onChange={(isBookmarked) => { handleIsBookmarked((option as TagShape | Tag).tag, isBookmarked); }}
                    />
                </MenuItem>
            )}
            renderInput={(params) => (
                <TextInput
                    value={inputValue}
                    onChange={onChange}
                    isOptional={isOptional}
                    label={t("Tag", { count: 2 })}
                    placeholder={placeholder ?? t("TagSelectorPlaceholder")}
                    InputProps={{
                        ...params.InputProps,
                        startAdornment: (
                            <>
                                <InputAdornment position="start">
                                    <TagIcon />
                                </InputAdornment>
                                {params.InputProps.startAdornment}
                            </>
                        ),
                        endAdornment: (
                            <>
                                {loading ? <CircularProgress color="inherit" size={20} /> : null}
                                {params.InputProps.endAdornment}
                            </>
                        ),
                    }}
                    inputProps={params.inputProps}
                    onKeyDown={onKeyDown}
                    fullWidth
                    sx={{ paddingRight: 0, minWidth: "250px" }}
                />
            )}
            sx={sx}
        />
    );
};

export const TagSelector = ({
    name,
    ...props
}: TagSelectorProps) => {
    const [field, , helpers] = useField<(TagShape | Tag)[] | undefined>(name);

    const handleTagsUpdate = useCallback((tags: (TagShape | Tag)[]) => {
        exists(helpers) && helpers.setValue(tags);
    }, [helpers]);

    return (
        <TagSelectorBase
            handleTagsUpdate={handleTagsUpdate}
            tags={field.value ?? []}
            {...props}
        />
    );
};

