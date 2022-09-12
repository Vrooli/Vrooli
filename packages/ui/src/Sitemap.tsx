import { DynamicSitemap } from "@shared/route";
import { Routes } from "Routes";
import { CommonProps } from "types";

// Generate Sitemap
export const Sitemap = (props: CommonProps) => <DynamicSitemap Routes={Routes} {...props} />