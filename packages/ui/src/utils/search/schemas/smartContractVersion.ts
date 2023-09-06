import { endpointGetSmartContractVersion, endpointGetSmartContractVersions, SmartContractVersionSortBy } from "@local/shared";
import { FormSchema } from "forms/types";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";

export const smartContractVersionSearchSchema = (): FormSchema => ({
    formLayout: searchFormLayout("SearchSmartContractVersion"),
    containers: [
        isCompleteWithRootContainer,
        isLatestContainer,
        votesRootContainer(),
        bookmarksRootContainer(),
        languagesContainer(),
        tagsRootContainer(),
    ],
    fields: [
        ...isCompleteWithRootFields(),
        ...isLatestFields(),
        ...votesRootFields(),
        ...bookmarksRootFields(),
        ...languagesFields(),
        ...tagsRootFields(),
    ],
});

export const smartContractVersionSearchParams = () => toParams(smartContractVersionSearchSchema(), endpointGetSmartContractVersions, endpointGetSmartContractVersion, SmartContractVersionSortBy, SmartContractVersionSortBy.DateCreatedDesc);
