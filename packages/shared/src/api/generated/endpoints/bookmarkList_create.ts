import gql from "graphql-tag";
import { Api_list } from "../fragments/Api_list";
import { Api_nav } from "../fragments/Api_nav";
import { Bookmark_full } from "../fragments/Bookmark_full";
import { Comment_common } from "../fragments/Comment_common";
import { Issue_nav } from "../fragments/Issue_nav";
import { Label_full } from "../fragments/Label_full";
import { Label_list } from "../fragments/Label_list";
import { Note_list } from "../fragments/Note_list";
import { Note_nav } from "../fragments/Note_nav";
import { Organization_list } from "../fragments/Organization_list";
import { Organization_nav } from "../fragments/Organization_nav";
import { Post_list } from "../fragments/Post_list";
import { Project_list } from "../fragments/Project_list";
import { Project_nav } from "../fragments/Project_nav";
import { Question_list } from "../fragments/Question_list";
import { QuestionAnswer_list } from "../fragments/QuestionAnswer_list";
import { Quiz_list } from "../fragments/Quiz_list";
import { Routine_list } from "../fragments/Routine_list";
import { Routine_nav } from "../fragments/Routine_nav";
import { SmartContract_list } from "../fragments/SmartContract_list";
import { SmartContract_nav } from "../fragments/SmartContract_nav";
import { Standard_list } from "../fragments/Standard_list";
import { Standard_nav } from "../fragments/Standard_nav";
import { Tag_list } from "../fragments/Tag_list";
import { User_list } from "../fragments/User_list";
import { User_nav } from "../fragments/User_nav";

export const bookmarkListCreate = gql`${Api_list}
${Api_nav}
${Bookmark_full}
${Comment_common}
${Issue_nav}
${Label_full}
${Label_list}
${Note_list}
${Note_nav}
${Organization_list}
${Organization_nav}
${Post_list}
${Project_list}
${Project_nav}
${Question_list}
${QuestionAnswer_list}
${Quiz_list}
${Routine_list}
${Routine_nav}
${SmartContract_list}
${SmartContract_nav}
${Standard_list}
${Standard_nav}
${Tag_list}
${User_list}
${User_nav}

mutation bookmarkListCreate($input: BookmarkListCreateInput!) {
  bookmarkListCreate(input: $input) {
    bookmarks {
        ...Bookmark_full
    }
    id
    created_at
    updated_at
    label
    bookmarksCount
  }
}`;

