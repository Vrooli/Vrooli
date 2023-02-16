import { SmartContractVersionSortBy } from "@shared/consts";
import { smartContractVersionFindMany } from "api/generated/endpoints/smartContractVersion";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const smartContractVersionSearchSchema = (lng: string): FormSchema => ({
    formLayout: searchFormLayout('SearchSmartContractVersion', lng),
    containers: [
        isCompleteWithRootContainer,
        isLatestContainer,
        votesRootContainer,
        bookmarksRootContainer,
        languagesContainer,
        tagsRootContainer,
    ],
    fields: [
        ...isCompleteWithRootFields,
        ...isLatestFields,
        ...votesRootFields,
        ...bookmarksRootFields,
        ...languagesFields,
        ...tagsRootFields,
    ]
})

export const smartContractVersionSearchParams = (lng: string) => toParams(smartContractVersionSearchSchema(lng), smartContractVersionFindMany, SmartContractVersionSortBy, SmartContractVersionSortBy.DateCreatedDesc);