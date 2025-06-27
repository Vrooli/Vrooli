import { useContext } from "react";
import { SessionContext } from "../contexts/session.js";

/**
 * Hook to check if the current user has admin privileges
 * @returns Object containing admin status and related data
 */
export function useIsAdmin() {
    const session = useContext(SessionContext);

    // Check if current user is admin
    // This checks if the current user ID matches the admin ID from the seeded data
    const isAdmin = Boolean(
        session.isLoggedIn &&
        session.users &&
        session.users.length > 0 &&
        session.users.some(user => user.isAdmin === true),
    );

    // Get admin user data if available
    const adminUser = session.users?.find(user => user.isAdmin === true);

    return {
        isAdmin,
        loading: session.loading,
        adminUser: isAdmin ? adminUser : null,
        userId: session.users?.[0]?.id || null,
    };
}

/**
 * Hook to check if the current user has specific admin permissions
 * This can be extended later for more granular admin permissions
 */
export function useAdminPermissions() {
    const { isAdmin, adminUser } = useIsAdmin();

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
