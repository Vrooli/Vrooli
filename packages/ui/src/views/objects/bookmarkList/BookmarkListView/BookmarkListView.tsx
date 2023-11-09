import { Bookmark, BookmarkCreateInput, BookmarkList, endpointGetBookmarkList, endpointPostBookmark, LINKS, uuid } from "@local/shared";
import { Box, IconButton, useTheme } from "@mui/material";
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
import { ObjectAction } from "utils/actions/objectActions";
import { listToAutocomplete } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getUserLanguages } from "utils/display/translationTools";
import { HistoryPageTabOption } from "utils/search/objectToSearch";
import { deleteArrayIndex, updateArray } from "utils/shape/general";
import { shapeBookmark } from "utils/shape/models/bookmark";
import { BookmarkListViewProps } from "../types";

export const BookmarkListView = ({
    display,
    isOpen,
    onClose,
}: BookmarkListViewProps) => {
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
            else setLocation(`${LINKS.History}?type=${HistoryPageTabOption.Bookmarked}`, { replace: true });
        },
    });

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
                options={[{
                    Icon: DeleteIcon,
                    label: t("Delete"),
                    onClick: handleDelete,
                }]}
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
                    dummyItems={new Array(5).fill("Routine")}
                    items={bookmarks}
                    keyPrefix="bookmark-list-item"
                    loading={isLoading}
                    onAction={onAction}
                />
            </ListContainer>
            <SideActionsButtons display={display} >
                <IconButton aria-label={t("UpdateList")} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} sx={{ background: palette.secondary.main }}>
                    <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
                <IconButton aria-label={t("AddBookmark")} onClick={openSearch} sx={{ background: palette.secondary.main }}>
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
            </SideActionsButtons>
        </>
    );
};
