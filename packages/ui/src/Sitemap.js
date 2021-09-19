import React from 'react';
import { Routes } from "Routes";
import DynamicSitemap from "react-dynamic-sitemap";

export function Sitemap(props) {
	return (
		<DynamicSitemap routes={Routes} prettify={true} {...props}/>
	);
}