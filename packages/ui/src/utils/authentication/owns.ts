import { MemberRole } from "@local/shared";

/**
 * Determines if a user owns/can edit a piece of content.
 * @param role Role to check
 * @returns True if user can edit, false if not
 */
export const owns = (role?: string | MemberRole | null | undefined): boolean => {
    if (!role) return false;
    return [MemberRole.Admin, MemberRole.Owner].includes(role as MemberRole);
}
