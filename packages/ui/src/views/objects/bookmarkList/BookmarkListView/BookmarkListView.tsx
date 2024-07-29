import { Bookmark, BookmarkCreateInput, BookmarkList, deleteArrayIndex, endpointGetBookmarkList, endpointPostBookmark, HistoryPageTabOption, LINKS, shapeBookmark, updateArray, uuid } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SiteSearchBar } from "components/inputs/search";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { ObjectListActions } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { useDeleter } from "hooks/useDeleter";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { AddIcon, DeleteIcon, EditIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { SideActionsButton } from "styles";
import { ObjectAction } from "utils/actions/objectActions";
import { DUMMY_LIST_LENGTH } from "utils/consts";
import { listToAutocomplete } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getUserLanguages } from "utils/display/translationTools";
import { BookmarkListViewProps } from "../types";

export function BookmarkListView({
    display,
    isOpen,
    onClose,
}: BookmarkListViewProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setBookmarkList } = useObjectFromUrl<BookmarkList>({
        ...endpointGetBookmarkList,
        objectType: "BookmarkList",
    });

    const { label } = useMemo(() => ({ label: existing?.label ?? "" }), [existing]);

    const [bookmarks, setBookmarks] = useState<Bookmark[]>([]);
    useEffect(() => {
        setBookmarks(existing?.bookmarks ?? []);
    }, [existing?.bookmarks]);

    const onAction = useCallback((action: keyof ObjectListActions<Bookmark>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                const id = data[0] as string;
                setBookmarks(curr => deleteArrayIndex(curr, curr.findIndex(item => item.id === id)));
                break;
            }
            case "Updated": {
                const updated = data[0] as Bookmark;
                setBookmarks(curr => updateArray(curr, curr.findIndex(item => item.id === updated.id), updated));
                break;
            }
        }
    }, []);

    const actionData = useObjectActions({
        object: existing,
        objectType: "BookmarkList",
        onAction,
        setLocation,
        setObject: setBookmarkList,
    });

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

    const {
        handleDelete,
        DeleteDialogComponent,
    } = useDeleter({
        object: existing,
        objectType: "BookmarkList",
        onActionComplete: () => {
            const hasPreviousPage = Boolean(sessionStorage.getItem("lastPath"));
            if (hasPreviousPage) window.history.back();
            else setLocation(`${LINKS.History}?type="${HistoryPageTabOption.Bookmarked}"`, { replace: true });
        },
    });

    const topBarOptions = useMemo(function topBarOptionsMemo() {
        return [{
            Icon: DeleteIcon,
            label: t("Delete"),
            onClick: handleDelete,
        }];
    }, [handleDelete, t]);

    return (
        <>
            {DeleteDialogComponent}
            <FindObjectDialog
                find="List"
                isOpen={searchOpen}
                handleCancel={closeSearch}
                handleComplete={closeSearch}
            />
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(label, t("BookmarkList", { count: 1 }))}
                options={topBarOptions}
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
                    />
                </Box>}
            />
            <ListContainer
                emptyText={t("NoResults", { ns: "error" })}
                isEmpty={bookmarks.length === 0 && !isLoading}
            >
                <ObjectList
                    dummyItems={new Array(DUMMY_LIST_LENGTH).fill("Routine")}
                    items={bookmarks}
                    keyPrefix="bookmark-list-item"
                    loading={isLoading}
                    onAction={onAction}
                />
            </ListContainer>
            <SideActionsButtons display={display} >
                <SideActionsButton aria-label={t("UpdateList")} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }}>
                    <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton>
                <SideActionsButton aria-label={t("AddBookmark")} onClick={openSearch}>
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton>
            </SideActionsButtons>
        </>
    );
}
