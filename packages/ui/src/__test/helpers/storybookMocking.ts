/**
 * Helper utilities for constructing MSW mock URLs in Storybook stories
 */

import { getObjectUrl } from "@vrooli/shared";
import { API_URL } from "../storybookConsts.js";

/**
 * Constructs a complete MSW mock URL for API endpoints
 * @param endpointConfig - The endpoint configuration from endpointsResource or other endpoint objects
 * @returns The complete URL path for MSW to mock
 * 
 * @example
 * // Instead of: `${API_URL}/v2${endpointsResource.findApiVersion.endpoint}`
 * // Use: getMockUrl(endpointsResource.findApiVersion)
 */
export function getMockUrl(endpointConfig: { endpoint: string }): string {
    return `${API_URL}/v2${endpointConfig.endpoint}`;
}

/**
 * Constructs a complete MSW mock URL for nested endpoint configurations
 * @param endpointConfig - The nested endpoint configuration (e.g., with findOne property)
 * @returns The complete URL path for MSW to mock
 * 
 * @example
 * // Instead of: `${API_URL}/v2${endpointsResource.findDataStructureVersion.findOne.endpoint}`
 * // Use: getMockUrlNested(endpointsResource.findDataStructureVersion.findOne)
 */
export function getMockUrlNested(endpointConfig: { endpoint: string }): string {
    return `${API_URL}/v2${endpointConfig.endpoint}`;
}

/**
 * Generic helper that handles both simple and nested endpoint configurations
 * @param endpointOrNested - Either a direct endpoint config or a nested one
 * @returns The complete URL path for MSW to mock
 * 
 * @example
 * // Works with direct endpoints:
 * // getMockEndpoint(endpointsResource.findApiVersion)
 * 
 * // Works with nested endpoints:
 * // getMockEndpoint(endpointsResource.findDataStructureVersion.findOne)
 */
export function getMockEndpoint(endpointOrNested: { endpoint: string } | { findOne: { endpoint: string } }): string {
    if ('endpoint' in endpointOrNested) {
        return `${API_URL}/v2${endpointOrNested.endpoint}`;
    }
    return `${API_URL}/v2${endpointOrNested.findOne.endpoint}`;
}

/**
 * Constructs a route path for Storybook story parameters
 * @param mockData - The mock data object that has an id or other properties for URL generation
 * @returns The complete route path for Storybook story parameters
 * 
 * @example
 * // Instead of: `${API_URL}/v2${getObjectUrl(mockApiVersionData)}`
 * // Use: getStoryRoutePath(mockApiVersionData)
 */
export function getStoryRoutePath(mockData: any): string {
    return `${API_URL}/v2${getObjectUrl(mockData)}`;
}

/**
 * Constructs a route path for edit pages in Storybook story parameters
 * @param mockData - The mock data object that has an id or other properties for URL generation
 * @returns The complete route path with /edit suffix for Storybook story parameters
 * 
 * @example
 * // Instead of: `${API_URL}/v2${getObjectUrl(mockApiVersionData)}/edit`
 * // Use: getStoryRouteEditPath(mockApiVersionData)
 */
export function getStoryRouteEditPath(mockData: any): string {
    return `${API_URL}/v2${getObjectUrl(mockData)}/edit`;
}

/**
 * Constructs a route path with query parameters for Storybook story parameters
 * @param mockData - The mock data object that has an id or other properties for URL generation
 * @param queryParams - Object containing query parameters to append
 * @returns The complete route path with query parameters for Storybook story parameters
 * 
 * @example
 * // Instead of: `${API_URL}/v2${getObjectUrl(mockData)}?runId=${generatePK().toString()}`
 * // Use: getStoryRoutePathWithQuery(mockData, { runId: generatePK().toString() })
 */
export function getStoryRoutePathWithQuery(mockData: any, queryParams: Record<string, string>): string {
    const baseUrl = `${API_URL}/v2${getObjectUrl(mockData)}`;
    const queryString = new URLSearchParams(queryParams).toString();
    return `${baseUrl}?${queryString}`;
}

/**
 * Constructs a direct API endpoint URL for MSW mocking
 * @param endpoint - The endpoint path (without /v2 prefix)
 * @returns The complete URL for MSW to mock
 * 
 * @example
 * // Instead of: `${API_URL}/v2/profile`
 * // Use: getMockApiUrl('/profile')
 * 
 * // Instead of: `${API_URL}/v2/admin/settings`
 * // Use: getMockApiUrl('/admin/settings')
 */
export function getMockApiUrl(endpoint: string): string {
    // Ensure endpoint starts with /
    const normalizedEndpoint = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
    return `${API_URL}/v2${normalizedEndpoint}`;
}