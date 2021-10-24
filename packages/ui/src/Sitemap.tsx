import { Routes } from "Routes";
import DynamicSitemap from "react-dynamic-sitemap";

// Define custom properties on Route object
declare module 'react-router-dom' {
    interface RouteProps {
        sitemapIndex?: boolean;
        priority?: number;
        changefreq?: string;
    }
}

// Generate Sitemap
export function Sitemap(props: any) {
	return (
		<DynamicSitemap routes={Routes} prettify={true} {...props}/>
	);
}