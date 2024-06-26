import { BookmarkList } from "@local/shared";
import { FormProps } from "forms/types";
import { BookmarkListShape } from "utils/shape/models/bookmarkList";
import { ObjectViewProps } from "views/types";
import { CrudPropsDialog, CrudPropsPage } from "../types";

type BookmarkListUpsertPropsPage = CrudPropsPage;
type BookmarkListUpsertPropsDialog = CrudPropsDialog<BookmarkList>;
export type BookmarkListUpsertProps = BookmarkListUpsertPropsPage | BookmarkListUpsertPropsDialog;
export type BookmarkListFormProps = FormProps<BookmarkList, BookmarkListShape>
export type BookmarkListViewProps = ObjectViewProps<BookmarkList>
