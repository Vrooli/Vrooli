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
        votesRootContainer(lng),
        bookmarksRootContainer(lng),
        languagesContainer(lng),
        tagsRootContainer(lng),
    ],
    fields: [
        ...isCompleteWithRootFields(lng),
        ...isLatestFields(lng),
        ...votesRootFields(lng),
        ...bookmarksRootFields(lng),
        ...languagesFields(lng),
        ...tagsRootFields(lng),
    ]
})

export const smartContractVersionSearchParams = (lng: string) => toParams(smartContractVersionSearchSchema(lng), smartContractVersionFindMany, SmartContractVersionSortBy, SmartContractVersionSortBy.DateCreatedDesc);