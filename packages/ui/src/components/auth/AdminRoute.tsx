import React from "react";
import { Navigate, useLocation } from "react-router-dom";
import { useIsAdmin } from "../../hooks/useIsAdmin.js";
import { PageLoader } from "../Page/PageLoader.js";
import { AlertBox } from "../AlertBox/AlertBox.js";

interface AdminRouteProps {
    children: React.ReactNode;
    /**
     * Custom fallback component to render when user is not admin
     * If not provided, redirects to home page
     */
    fallback?: React.ReactNode;
    /**
     * Whether to show a loading state while checking admin status
     * @default true
     */
    showLoading?: boolean;
}

/**
 * Route protection component that only allows admin users to access wrapped content
 * 
 * Usage:
 * ```tsx
 * <AdminRoute>
 *   <AdminDashboard />
 * </AdminRoute>
 * ```
 */
export const AdminRoute: React.FC<AdminRouteProps> = ({ 
    children, 
    fallback,
    showLoading = true 
}) => {
    const { isAdmin, loading } = useIsAdmin();
    const location = useLocation();
    
    // Show loading state while checking admin status
    if (loading && showLoading) {
        return <PageLoader />;
    }
    
    // If user is not admin, show fallback or redirect
    if (!isAdmin) {
        if (fallback) {
            return <>{fallback}</>;
        }
        
        // Redirect to home page, preserving the intended destination in state
        return (
            <Navigate 
                to="/" 
                state={{ from: location }} 
                replace 
            />
        );
    }
    
    // User is admin, render children
    return <>{children}</>;
};

/**
 * Component to display when user doesn't have admin access
 */
export const AdminAccessDenied: React.FC = () => {
    return (
        <AlertBox 
            severity="error"
            title="Access Denied"
            sx={{ m: 2 }}
        >
            You need administrator privileges to access this page. 
            Please contact your system administrator if you believe this is an error.
        </AlertBox>
    );
};

/**
 * Higher-order component version of AdminRoute for class components
 */
export const withAdminProtection = <P extends object>(
    Component: React.ComponentType<P>
) => {
    const ProtectedComponent: React.FC<P> = (props) => (
        <AdminRoute>
            <Component {...props} />
        </AdminRoute>
    );
    
    ProtectedComponent.displayName = `withAdminProtection(${Component.displayName || Component.name})`;
    
    return ProtectedComponent;
};