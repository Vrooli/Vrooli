import { BookmarkList } from "@local/shared";
import { FormProps } from "forms/types";
import { BookmarkListShape } from "utils/shape/models/bookmarkList";
import { ObjectViewProps } from "views/types";
import { UpsertProps } from "../types";

export type BookmarkListUpsertProps = UpsertProps<BookmarkList>
export type BookmarkListFormProps = FormProps<BookmarkList, BookmarkListShape>
export type BookmarkListViewProps = ObjectViewProps<BookmarkList>
