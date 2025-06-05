import { Autocomplete, Chip, CircularProgress, InputAdornment, ListItemText, MenuItem, Popper, styled, type AutocompleteRenderGetTagProps, type AutocompleteRenderInputParams, type PopperProps } from "@mui/material";
import { BookmarkFor, DUMMY_ID, TagSortBy, endpointsTag, exists, type Tag, type TagSearchInput, type TagSearchResult, type TagShape } from "@vrooli/shared";
import { useField } from "formik";
import { useCallback, useEffect, useMemo, useRef, useState, type HTMLAttributes } from "react";
import { useTranslation } from "react-i18next";
import { useFetch } from "../../../hooks/useFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { CHIP_LIST_LIMIT } from "../../../utils/consts.js";
import { PubSub } from "../../../utils/pubsub.js";
import { BookmarkButton } from "../../buttons/BookmarkButton.js";
import { TextInput } from "../TextInput/TextInput.js";
import { type TagSelectorBaseProps, type TagSelectorProps } from "../types.js";

type TagOption = string | Tag | TagShape;

function filterOptions(options: TagOption[]) { return options; }

function isOptionEqualToValue(option: TagOption, value: TagOption) {
    const optionTag = typeof option === "string" ? option : option.tag;
    const valueTag = typeof value === "string" ? value : value.tag;
    return optionTag === valueTag;
}

function getOptionLabel(o: TagOption) {
    return typeof o === "string" ? o : o.tag;
}

/** Removes invalid characters from tag string */
function withoutInvalidChars(str: string) {
    return str.replace(/[,;]/g, "");
}

// Style for the outer container, mimicking AdvancedInput
const Outer = styled("div")(({ theme }) => ({
    backgroundColor: theme.palette.background.paper,
    borderRadius: theme.spacing(3),
    padding: theme.spacing(1),
    position: "relative",
    cursor: "text",
}));

/** Custom Popper component to add scroll handling */
function PopperComponent({
    onScrollBottom,
    ...props
}: PopperProps & { onScrollBottom: () => unknown }) {
    const popperRef = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        function handleScroll(event) {
            const target = event.target;
            // Check if we've scrolled to the bottom
            if (target.scrollTop + target.clientHeight >= target.scrollHeight - 10) {
                // Trigger load more function
                onScrollBottom();
            }
        }

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
}

const PAGE_SIZE = 25;
const MIN_TAG_LENGTH = 2;
const MAX_TAG_LENGTH = 30;

const TagChip = styled(Chip)(({ theme }) => ({
    backgroundColor: theme.palette.mode === "light" ? "#8148b0" : "#8148b0", //'#a068ce',
    color: "white",
}));

// Define the styled TextInput component
const StyledTextInput = styled(TextInput)(({ theme }) => ({
    "& .MuiInputBase-root": {
        padding: 0, // Apply padding 0 to the root of the input
        paddingLeft: theme.spacing(1),
        paddingRight: theme.spacing(1),
        backgroundColor: "transparent", // Ensure background is transparent
    },
    "& .MuiOutlinedInput-notchedOutline": {
        border: "none", // Remove the default outline/border
    },
    minWidth: "250px",
    // Apply caption styles to the placeholder
    "& .MuiInputBase-input::placeholder": {
        ...theme.typography.caption,
        // Optionally adjust opacity if needed, MUI default is often less than 1
        opacity: 0.6,
    },
}));

export function TagSelectorBase({
    disabled,
    handleTagsUpdate,
    isRequired = false,
    tags,
    placeholder,
    sx,
}: TagSelectorBaseProps) {
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

    const onChange = useCallback((change: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        const sanitized = withoutInvalidChars(change.target.value);
        setInputValue(sanitized);
    }, []);

    // Detect when the tag should be submitted
    const onKeyDown = useCallback((event: React.KeyboardEvent<HTMLDivElement>) => {
        // Check if the user pressed ',', ';', or enter
        if (!["Comma", "Semicolon", "Enter"].includes(event.code)) return;
        // Remove invalid characters
        const tagLabel = withoutInvalidChars(inputValue);
        // Check if tag is valid length
        if (tagLabel.length < MIN_TAG_LENGTH) {
            PubSub.get().publish("snack", { messageKey: "TagTooShort", severity: "Error" });
            return;
        }
        if (tagLabel.length > MAX_TAG_LENGTH) {
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
        setInputValue("");
    }, [handleTagAdd, inputValue, tags]);

    const onInputSelect = useCallback((tag: TagShape | Tag) => {
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
        ...endpointsTag.findMany,
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

    const handleIsBookmarked = useCallback((tag: string, isBookmarked: boolean) => {
        if (!tagsRef.current[tag]) return;
        (tagsRef.current[tag] as TagShape).you = {
            ...((tagsRef.current[tag] as TagShape).you ?? {}),
            __typename: "TagYou" as const,
            isBookmarked,
        } as Tag["you"];
    }, [tagsRef]);

    const handleAutocompleteChange = useCallback((_event: React.SyntheticEvent, newValue: TagOption[]) => {
        // Ensure newValue is always an array, even if cleared
        const newTags = Array.isArray(newValue) ? newValue : [];
        handleTagsUpdate(newTags as (TagShape | Tag)[]);
    }, [handleTagsUpdate]);

    const popperComponentRender = useCallback(function popperComponentRender(props: PopperProps) {
        return <PopperComponent {...props} onScrollBottom={loadMoreTags} />;
    }, [loadMoreTags]);

    const renderTags = useCallback(function renderTagsMemo(value: TagOption[], getTagProps: AutocompleteRenderGetTagProps) {
        return value.map((option, index) => {
            function handleDelete() {
                onChipDelete(option as TagShape | Tag);
            }

            return (
                <TagChip
                    {...getTagProps({ index })}
                    id={`tag-chip-${typeof option === "string" ? option : option.tag}`}
                    key={typeof option === "string" ? option : option.tag}
                    variant="filled"
                    label={typeof option === "string" ? option : option.tag}
                    onDelete={handleDelete}
                />
            );
        });
    }, [onChipDelete]);

    const renderOption = useCallback(function renderOptionCallback(props: HTMLAttributes<HTMLLIElement>, option: TagOption) {
        function handleClick() {
            if (typeof option === "string") {
                const found = autocompleteOptions.find(t => t.tag === option);
                if (found) onInputSelect(found);
            } else {
                onInputSelect(option);
            }
        }

        function handleBookmarkChange(isBookmarked: boolean) {
            handleIsBookmarked((option as TagShape | Tag).tag, isBookmarked);
        }

        return (
            <MenuItem
                {...props}
                onClick={handleClick}
                selected={tags.some(t => typeof option === "string" ? (t as unknown as string) === option : (t as TagShape | Tag).tag === option.tag)}
            >
                <ListItemText>{typeof option === "string" ? option : option.tag}</ListItemText>
                <BookmarkButton
                    objectId={(option as Tag).id ?? ""}
                    bookmarkFor={BookmarkFor.Tag}
                    isBookmarked={(option as Tag).you.isBookmarked}
                    bookmarks={(option as Tag).bookmarks}
                    onChange={handleBookmarkChange}
                />
            </MenuItem>
        );
    }, [autocompleteOptions, handleIsBookmarked, onInputSelect, tags]);

    const renderInput = useCallback(function renderInputCallback(params: AutocompleteRenderInputParams) {
        const inputProps = {
            ...params.InputProps,
            startAdornment: (
                <>
                    <InputAdornment position="start">
                        <IconCommon
                            decorative
                            name="Tag"
                        />
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
        } as const;

        return (
            <StyledTextInput // Use the new styled component
                value={inputValue}
                onChange={onChange}
                isRequired={isRequired}
                placeholder={placeholder ?? t("TagSelectorPlaceholder")}
                InputProps={inputProps}
                inputProps={params.inputProps}
                onKeyDown={onKeyDown}
                fullWidth
                // Remove the sx prop for inputTextStyle as minWidth is handled by StyledTextInput
                // sx={inputTextStyle}
                // Remove the default variant which adds the border
                variant="outlined" // Keep outlined for structure but border is removed via StyledTextInput
            />
        );
    }, [inputValue, isRequired, loading, onChange, onKeyDown, placeholder, t]);

    return (
        // Wrap the Autocomplete with the styled Outer component
        <Outer sx={sx}>
            <Autocomplete
                id="tags-input"
                disabled={disabled}
                disablePortal
                fullWidth
                multiple
                // Allow all options through the filter - we perform custom filtering
                filterOptions={filterOptions}
                freeSolo={true}
                isOptionEqualToValue={isOptionEqualToValue}
                options={autocompleteOptions}
                getOptionLabel={getOptionLabel}
                inputValue={inputValue}
                noOptionsText={t("NoSuggestions")}
                limitTags={CHIP_LIST_LIMIT}
                loading={loading}
                value={tags}
                defaultValue={tags}
                PopperComponent={popperComponentRender}
                renderTags={renderTags}
                renderOption={renderOption}
                renderInput={renderInput}
                onChange={handleAutocompleteChange}
            // Remove sx from Autocomplete itself as it's now on the Outer wrapper
            // sx={sx}
            />
        </Outer>
    );
}

export function TagSelector({
    name,
    ...props
}: TagSelectorProps) {
    const [field, , helpers] = useField<(TagShape | Tag)[] | undefined>(name);

    const tags = useMemo(() => field.value ?? [], [field.value]);
    const handleTagsUpdate = useCallback((tags: (TagShape | Tag)[]) => {
        exists(helpers) && helpers.setValue(tags);
    }, [helpers]);

    return (
        <TagSelectorBase
            handleTagsUpdate={handleTagsUpdate}
            tags={tags}
            {...props}
        />
    );
}

