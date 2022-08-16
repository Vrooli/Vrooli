import { OrganizationView } from "components";
import { OrganizationViewPageProps } from "./types";
import { OrganizationCreate } from "components/views/OrganizationCreate/OrganizationCreate";
import { OrganizationUpdate } from "components/views/OrganizationUpdate/OrganizationUpdate";
import { ObjectPage } from "pages";

export const OrganizationViewPage = ({
    session
}: OrganizationViewPageProps) => {
    return (
        <ObjectPage
            session={session}
            Create={OrganizationCreate}
            Update={OrganizationUpdate}
            View={OrganizationView}
        />
    )
}