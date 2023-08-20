import { Bookmark, BookmarkList, bookmarkListValidation, DUMMY_ID, Session, uuid } from "@local/shared";
import { Box, Button, IconButton, List, ListItem, ListItemText, Stack, TextField, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Field, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { BookmarkListFormProps } from "forms/types";
import { AddIcon, DeleteIcon } from "icons";
import { forwardRef, useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis } from "styles";
import { getDisplay } from "utils/display/listTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { BookmarkShape } from "utils/shape/models/bookmark";
import { BookmarkListShape, shapeBookmarkList } from "utils/shape/models/bookmarkList";

export const bookmarkListInitialValues = (
    session: Session | undefined,
    existing?: Partial<BookmarkList> | null | undefined,
): BookmarkListShape => ({
    __typename: "BookmarkList" as const,
    id: DUMMY_ID,
    label: "",
    bookmarks: [],
    ...existing,
});

export const transformBookmarkListValues = (values: BookmarkListShape, existing: BookmarkListShape, isCreate: boolean) =>
    isCreate ? shapeBookmarkList.create(values) : shapeBookmarkList.update(existing, values);

export const validateBookmarkListValues = async (values: BookmarkListShape, existing: BookmarkListShape, isCreate: boolean) => {
    const transformedValues = transformBookmarkListValues(values, existing, isCreate);
    const validationSchema = bookmarkListValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const BookmarkListForm = forwardRef<BaseFormRef | undefined, BookmarkListFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    ...props
}, ref) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [idField, , idHelpers] = useField<string>("id");
    const [bookmarksField, bookmarksMeta, bookmarksHelpers] = useField<BookmarkShape[]>("bookmarks");

    const addNewBookmark = useCallback((to: BookmarkShape) => {
        bookmarksHelpers.setValue([...bookmarksField.value, {
            __typename: "Bookmark" as const,
            id: uuid(),
            to,
            list: { __typename: "BookmarkList", id: idField.value },
        } as any]);
    }, [bookmarksField, bookmarksHelpers, idField]);

    const removeBookmark = useCallback((index: number) => {
        const newBookmarks = [...bookmarksField.value];
        newBookmarks.splice(index, 1);
        bookmarksHelpers.setValue(newBookmarks);
    }, [bookmarksField, bookmarksHelpers]);

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

    return (
        <>
            <FindObjectDialog
                find="List"
                isOpen={searchOpen}
                handleCancel={closeSearch}
                handleComplete={closeSearch}
            />
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={700}
                ref={ref}
            >
                <Stack direction="column" spacing={2} m={2} mb={4}>
                    <Box mb={2} mt={2}>
                        <Field
                            fullWidth
                            name="label"
                            label={t("Label")}
                            as={TextField}
                        />
                    </Box>
                    {bookmarksField.value.length ? <List>
                        {bookmarksField.value.map((bookmark, index) => (
                            <ListItem
                                key={bookmark.id}
                                disablePadding
                                component={"div"}
                                sx={{
                                    display: "flex",
                                    background: palette.background.paper,
                                    padding: "8px 16px",
                                    borderBottom: `1px solid ${palette.divider}`,
                                }}
                            >
                                <Stack
                                    direction="column"
                                    spacing={1}
                                    pl={2}
                                    sx={{
                                        width: "-webkit-fill-available",
                                        display: "grid",
                                        pointerEvents: "none",
                                    }}
                                >
                                    <ListItemText
                                        primary={getDisplay(bookmark as Bookmark).title}
                                        sx={{
                                            ...multiLineEllipsis(1),
                                            lineBreak: "anywhere",
                                            pointerEvents: "none",
                                        }}
                                    />
                                    <MarkdownDisplay
                                        content={getDisplay(bookmark as Bookmark).subtitle}
                                        sx={{ ...multiLineEllipsis(2), color: palette.text.secondary, pointerEvents: "none" }}
                                    />
                                </Stack>
                                <IconButton
                                    onClick={() => removeBookmark(index)}
                                    sx={{
                                        justifySelf: "end",
                                        pointerEvents: "all",
                                    }}
                                >
                                    <DeleteIcon fill={palette.secondary.main} />
                                </IconButton>
                            </ListItem>
                        ))}
                    </List> : null}
                    <Button
                        startIcon={<AddIcon />}
                        onClick={openSearch}
                        variant="outlined"
                        sx={{ alignSelf: "center" }}
                    >
                        Add Bookmark
                    </Button>
                </Stack>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={props.errors}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});
