import { SmartContractVersionSortBy } from "@local/consts";
import { smartContractVersionFindMany } from "../../../api/generated/endpoints/smartContractVersion_findMany";
import { toParams } from "./base";
import { bookmarksRootContainer, bookmarksRootFields, isCompleteWithRootContainer, isCompleteWithRootFields, isLatestContainer, isLatestFields, languagesContainer, languagesFields, searchFormLayout, tagsRootContainer, tagsRootFields, votesRootContainer, votesRootFields } from "./common";
export const smartContractVersionSearchSchema = () => ({
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
export const smartContractVersionSearchParams = () => toParams(smartContractVersionSearchSchema(), smartContractVersionFindMany, SmartContractVersionSortBy, SmartContractVersionSortBy.DateCreatedDesc);
//# sourceMappingURL=smartContractVersion.js.map