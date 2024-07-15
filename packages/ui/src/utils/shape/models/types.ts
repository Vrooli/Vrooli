export type OwnerShape = {
    __typename: "User" | "Team",
    id: string,
    handle?: string | null,
    name?: string
    profileImage?: string | null,
};
export type ParentShape = { id: string };
