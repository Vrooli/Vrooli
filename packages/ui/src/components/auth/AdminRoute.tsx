import React from "react";
import { Redirect, useLocation } from "../../route/router.js";
import { useIsAdmin } from "../../hooks/useIsAdmin.js";
import { DiagonalWaveLoader } from "../Spinners.js";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { LINKS } from "@vrooli/shared";

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
    const [{ pathname }] = useLocation();
    
    // Show loading state while checking admin status
    if (loading && showLoading) {
        return (
            <div style={{ 
                display: 'flex', 
                justifyContent: 'center', 
                alignItems: 'center', 
                minHeight: '200px' 
            }}>
                <DiagonalWaveLoader />
            </div>
        );
    }
    
    // If user is not admin, show fallback or redirect
    if (!isAdmin) {
        if (fallback) {
            return <>{fallback}</>;
        }
        
        // Redirect to home page
        return (
            <Redirect 
                to={LINKS.Home} 
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
        <Box 
            sx={{ 
                m: 2, 
                p: 2, 
                border: '1px solid',
                borderColor: 'error.main',
                borderRadius: 1,
                backgroundColor: 'error.light',
                color: 'error.contrastText'
            }}
        >
            <Typography variant="h6" component="h2" gutterBottom>
                Access Denied
            </Typography>
            <Typography variant="body2">
                You need administrator privileges to access this page. 
                Please contact your system administrator if you believe this is an error.
            </Typography>
        </Box>
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