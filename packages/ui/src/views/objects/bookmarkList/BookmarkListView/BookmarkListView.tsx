import { Bookmark, BookmarkCreateInput, BookmarkList, endpointGetBookmarkList, endpointPostBookmark, uuid } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SiteSearchBar } from "components/inputs/search";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { AddIcon, EditIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ObjectAction } from "utils/actions/objectActions";
import { listToAutocomplete } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { firstString } from "utils/display/stringTools";
import { getUserLanguages } from "utils/display/translationTools";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SessionContext } from "utils/SessionContext";
import { shapeBookmark } from "utils/shape/models/bookmark";
import { BookmarkListViewProps } from "../types";

export const BookmarkListView = ({
    isOpen,
    onClose,
    zIndex,
}: BookmarkListViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const display = toDisplay(isOpen);

    const { object: existing, isLoading, setObject: setBookmarkList } = useObjectFromUrl<BookmarkList>({
        ...endpointGetBookmarkList,
        objectType: "BookmarkList",
    });

    const { label } = useMemo(() => ({ label: existing?.label ?? "" }), [existing]);

    const actionData = useObjectActions({
        object: existing,
        objectType: "BookmarkList",
        setLocation,
        setObject: setBookmarkList,
    });

    const [bookmarks, setBookmarks] = useState<Bookmark[]>([]);
    useEffect(() => {
        setBookmarks(existing?.bookmarks ?? []);
    }, [existing?.bookmarks]);

    const [createBookmark, { loading: isCreating, errors: createErrors }] = useLazyFetch<BookmarkCreateInput, Bookmark>(endpointPostBookmark);
    const addNewBookmark = useCallback(async (to: any) => {
        fetchLazyWrapper<BookmarkCreateInput, Bookmark>({
            fetch: createBookmark,
            inputs: shapeBookmark.create({
                __typename: "Bookmark" as const,
                id: uuid(),
                to,
                list: { __typename: "BookmarkList", id: existing?.id ?? "" },
            }),
            successCondition: (data) => data !== null,
            onSuccess: (data) => {
                setBookmarks((prev) => [...prev, data]);
            },
        });
    }, [createBookmark, existing?.id]);

    // Search dialog to find objects to bookmark
    const hasSelectedObject = useRef(false);
    const [searchOpen, setSearchOpen] = useState(false);
    const openSearch = useCallback(() => { setSearchOpen(true); }, []);
    const closeSearch = useCallback((selectedObject?: any) => {
        setSearchOpen(false);
        hasSelectedObject.current = !!selectedObject;
        if (selectedObject) {
            addNewBookmark(selectedObject);
        }
    }, [addNewBookmark]);

    const [searchString, setSearchString] = useState("");
    const updateSearchString = useCallback((newString: string) => {
        setSearchString(newString);
    }, []);

    const onBookmarkSelect = useCallback((data: any) => {
        console.log("onBookmarkSelect", data);
    }, []);

    const autocompleteOptions = useMemo(() => listToAutocomplete(bookmarks, getUserLanguages(session)), [bookmarks, session]);

    return (
        <>
            <FindObjectDialog
                find="List"
                isOpen={searchOpen}
                handleCancel={closeSearch}
                handleComplete={closeSearch}
                zIndex={zIndex + 1}
            />
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(label, t("BookmarkList", { count: 1 }))}
                below={<Box sx={{
                    width: "min(100%, 700px)",
                    margin: "auto",
                    marginTop: 2,
                    marginBottom: 2,
                    paddingLeft: 2,
                    paddingRight: 2,
                }}>
                    <SiteSearchBar
                        id={`${existing?.id ?? "bookmark-list"}-search-bar`}
                        placeholder={"SearchBookmark"}
                        loading={false}
                        value={searchString}
                        onChange={updateSearchString}
                        onInputChange={onBookmarkSelect}
                        options={autocompleteOptions}
                        sxs={{ root: { width: "min(100%, 600px)", paddingLeft: 2, paddingRight: 2 } }}
                        zIndex={zIndex}
                    />
                </Box>}
                zIndex={zIndex}
            />
            <>
                <SideActionButtons display={display} zIndex={zIndex + 1}>
                    {/* Edit button */}
                    <ColorIconButton aria-label="Edit list" background={palette.secondary.main} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                    {/* Add button */}
                    <ColorIconButton aria-label="Add bookmark" background={palette.secondary.main} onClick={openSearch} >
                        <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                </SideActionButtons>
                <ListContainer
                    emptyText={t("NoResults", { ns: "error" })}
                    isEmpty={bookmarks.length === 0 && !isLoading}
                >
                    <ObjectList
                        dummyItems={new Array(5).fill("Routine")}
                        items={bookmarks}
                        keyPrefix="bookmark-list-item"
                        loading={isLoading}
                        zIndex={zIndex}
                    />
                </ListContainer>
            </>
        </>
    );
};
