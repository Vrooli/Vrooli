import { ListObject } from "@local/shared";
import { Box, CircularProgress, List, ListItem, Popover } from "@mui/material";
import { Dispatch, SetStateAction, useEffect, useState } from "react";
import { noSelect } from "styles";
import { GetTaggableItemsFunc } from "../types";

export interface RichInputTagDropdownData {
    anchorEl: HTMLElement | null;
    list: ListObject[];
    tabIndex: number;
    setAnchorEl: Dispatch<SetStateAction<HTMLElement | null>>;
    setTabIndex: Dispatch<SetStateAction<number>>;
    setList: Dispatch<SetStateAction<ListObject[]>>;
    setTagString: Dispatch<SetStateAction<string>>;
    tagString: string;
}

const anchorOrigin = {
    vertical: "top",
    horizontal: "left",
} as const;
const transformOrigin = {
    vertical: "bottom",
    horizontal: "left",
} as const;

export function useTagDropdown({
    getTaggableItems,
}: {
    getTaggableItems?: GetTaggableItemsFunc;
}): RichInputTagDropdownData {
    // Handle "@" dropdown for tagging users or any other items
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const [tabIndex, setTabIndex] = useState(0); // The index of the currently selected item in the dropdown
    const [tagString, setTagString] = useState(""); // What has been typed after the "@"
    const [list, setList] = useState<ListObject[]>([]); // The list of items currently being displayed
    const [cachedTags, setCachedTags] = useState<Record<string, ListObject[]>>({}); // The cached list of items for each tag string, to prevent unnecessary queries
    // Callback to parent to get items, or to get cached items if they exist
    useEffect(() => {
        if (!anchorEl || typeof getTaggableItems !== "function") return;

        async function fetchItems(getTaggableItems: GetTaggableItemsFunc) {
            // Check the cache
            if (cachedTags[tagString] !== undefined) {
                setList(cachedTags[tagString]);
            }
            // Fallback to calling the parent
            else {
                const items = await getTaggableItems(tagString);
                // Store the items in the cache
                setCachedTags(prev => ({ ...prev, [tagString]: items }));

                // If the list is empty, end the query
                if (items.length === 0) {
                    setAnchorEl(null);
                }
                // Otherwise, set the list
                else {
                    setList(items);
                }
            }
        }

        fetchItems(getTaggableItems);
    }, [cachedTags, anchorEl, getTaggableItems, tagString]);

    return {
        anchorEl,
        list,
        tabIndex,
        setAnchorEl,
        setList,
        setTabIndex,
        setTagString,
        tagString,
    };
}

/** When using "@" to tag an object, displays options */
export function RichInputTagDropdown({
    anchorEl,
    list,
    tabIndex,
    selectDropdownItem,
    setAnchorEl,
    setTabIndex,
}: RichInputTagDropdownData & {
    selectDropdownItem: (item: ListObject) => unknown;
}) {
    const [popoverWidth, setPopoverWidth] = useState<number | null>(null);

    useEffect(() => {
        if (anchorEl) {
            setPopoverWidth(anchorEl.offsetWidth);
        }
    }, [anchorEl]);

    const listItems = list.map((item, index) => (
        <ListItem
            key={item.id}
            selected={tabIndex === index}
            onMouseEnter={() => setTabIndex(index)}
            onClick={() => selectDropdownItem(item)}
            sx={{
                ...noSelect,
                cursor: "pointer",
            }}
        >
            {item.name}
        </ListItem>
    ));

    return (
        <Popover
            disableAutoFocus={true}
            open={Boolean(anchorEl)}
            anchorEl={anchorEl}
            onClose={() => setAnchorEl(null)}
            anchorOrigin={anchorOrigin}
            transformOrigin={transformOrigin}
            sx={{
                "& .MuiPopover-paper": {
                    width: popoverWidth,
                },
            }}
        >
            {list.length > 0 ? <List>
                {listItems}
            </List> :
                <Box sx={{
                    padding: 2,
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                }}>
                    <CircularProgress size={32} />
                </Box>
            }
        </Popover>
    );
}
