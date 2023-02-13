import { UserSortBy } from "@shared/consts";
import { userFindMany } from "api/generated/endpoints/user";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { languagesContainer, languagesFields, searchFormLayout, starsContainer, starsFields } from "./common";

export const userSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchUser', lng),
    containers: [
        starsContainer,
        languagesContainer,
    ],
    fields: [
        ...starsFields,
        ...languagesFields,
    ]
})

export const userSearchParams = (lng: string) => toParams(userSearchSchema(lng), userFindMany, UserSortBy, UserSortBy.StarsDesc)