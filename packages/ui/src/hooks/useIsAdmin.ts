import { SEEDED_PUBLIC_IDS } from "@vrooli/shared";
import { useContext } from "react";
import { SessionContext } from "../contexts/session.js";

/**
 * Hook to check if the current user has admin privileges
 * @returns Object containing admin status and related data
 */
export function useIsAdmin() {
    const session = useContext(SessionContext);

    // Handle case where session context is not yet initialized
    if (!session) {
        return {
            isAdmin: false,
            loading: true,
            adminUser: null,
            userId: null,
        };
    }

    // Check if current user is admin by comparing publicId with the seeded admin ID
    const currentUser = session.users?.[0];
    const isAdmin = Boolean(
        session.isLoggedIn &&
        currentUser &&
        currentUser.publicId === SEEDED_PUBLIC_IDS.Admin,
    );

    return {
        isAdmin,
        loading: false, // Session doesn't have a loading property
        adminUser: isAdmin ? currentUser : null,
        userId: currentUser?.id || null,
    };
}

/**
 * Hook to check if the current user has specific admin permissions
 * This can be extended later for more granular admin permissions
 */
export function useAdminPermissions() {
    const { isAdmin } = useIsAdmin();

    return {
        canManageUsers: isAdmin,
        canManageSystemSettings: isAdmin,
        canViewAnalytics: isAdmin,
        canManageApiKeys: isAdmin,
        canManageExternalServices: isAdmin,
        canAccessReports: isAdmin,
        // Future: more granular permissions based on adminUser roles
    };
}
