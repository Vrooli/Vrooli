import { Box, IconButton } from "@mui/material";
import { HistoryPageTabOption, LINKS, deleteArrayIndex, endpointsBookmark, shapeBookmark, updateArray, generatePK, type Bookmark, type BookmarkCreateInput, type BookmarkList } from "@vrooli/shared";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { SideActionsButtons } from "../../../components/buttons/SideActionsButtons.js";
import { ListContainer } from "../../../components/containers/ListContainer.js";
import { FindObjectDialog } from "../../../components/dialogs/FindObjectDialog/FindObjectDialog.js";
import { ObjectList } from "../../../components/lists/ObjectList/ObjectList.js";
import { type ObjectListActions } from "../../../components/lists/types.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { useDeleter, useObjectActions } from "../../../hooks/objectActions.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { DUMMY_LIST_LENGTH } from "../../../utils/consts.js";
import { listToAutocomplete } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { type BookmarkListViewProps } from "./types.js";

export function BookmarkListView({
    display,
    isOpen,
    onClose,
}: BookmarkListViewProps) {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const [{ pathname }, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setBookmarkList } = useManagedObject<BookmarkList>({ pathname });

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

    const [createBookmark, { loading: isCreating, errors: createErrors }] = useLazyFetch<BookmarkCreateInput, Bookmark>(endpointsBookmark.createOne);
    const addNewBookmark = useCallback(async (to: any) => {
        fetchLazyWrapper<BookmarkCreateInput, Bookmark>({
            fetch: createBookmark,
            inputs: shapeBookmark.create({
                __typename: "Bookmark" as const,
                id: generatePK(),
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
            iconInfo: { name: "Delete", type: "Common" } as const,
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
                below={<Box width="min(100%, 700px)" margin="auto" marginTop={2} marginBottom={2} paddingLeft={2} paddingRight={2}>
                    {/* <SiteSearchBarPaper
                        id={`${existing?.id ?? "bookmark-list"}-search-bar`}
                        placeholder={"SearchBookmark"}
                        loading={false}
                        value={searchString}
                        onChange={updateSearchString}
                        onInputChange={onBookmarkSelect}
                        options={autocompleteOptions}
                        sxs={{ root: { width: "min(100%, 600px)", paddingLeft: 2, paddingRight: 2 } }}
                    /> */}
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
                <IconButton aria-label={t("UpdateList")} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }}>
                    <IconCommon name="Edit" />
                </IconButton>
                <IconButton aria-label={t("AddBookmark")} onClick={openSearch}>
                    <IconCommon name="Add" />
                </IconButton>
            </SideActionsButtons>
        </>
    );
}
