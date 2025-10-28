import { resolveApiBase } from '@vrooli/api-base';
export const DEFAULT_API_PORT = import.meta.env.VITE_API_PORT?.trim() || '16430';
export const DEFAULT_API_TOKEN = import.meta.env.VITE_API_TOKEN?.trim() || 'math-tools-api-token';
export const DEFAULT_API_BASE_URL = resolveApiBase({
    explicitUrl: import.meta.env.VITE_API_BASE_URL?.trim(),
    defaultPort: DEFAULT_API_PORT,
    appendSuffix: false,
});
export async function apiRequest(path, options) {
    const { method = 'GET', body, headers, settings } = options;
    const normalizedBase = settings.baseUrl.replace(/\/+$/u, '');
    const normalizedPath = path.startsWith('http') ? path : `${normalizedBase}${path.startsWith('/') ? '' : '/'}${path}`;
    const requestInit = {
        method,
        headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${settings.token}`,
            ...headers,
        },
        body: body ? JSON.stringify(body) : undefined,
    };
    const response = await fetch(normalizedPath, requestInit);
    const contentType = response.headers.get('content-type') || '';
    const isJson = contentType.includes('application/json');
    const payload = isJson ? await response.json() : await response.text();
    if (!response.ok) {
        const message = typeof payload === 'string' ? payload : payload?.error || 'Unexpected API error';
        throw new Error(message);
    }
    return payload;
}
export function buildApiSettings(partial) {
    return {
        baseUrl: partial?.baseUrl?.trim() || DEFAULT_API_BASE_URL,
        token: partial?.token?.trim() || DEFAULT_API_TOKEN,
    };
}
export async function fetchHealth(settings) {
    return apiRequest('/api/health', { settings });
}
export async function performCalculation(payload, settings) {
    return apiRequest('/api/v1/math/calculate', { method: 'POST', body: payload, settings });
}
export async function runStatistics(payload, settings) {
    return apiRequest('/api/v1/math/statistics', {
        method: 'POST',
        body: payload,
        settings,
    });
}
export async function solveEquation(payload, settings) {
    return apiRequest('/api/v1/math/solve', {
        method: 'POST',
        body: payload,
        settings,
    });
}
export async function optimize(payload, settings) {
    return apiRequest('/api/v1/math/optimize', {
        method: 'POST',
        body: payload,
        settings,
    });
}
export async function forecast(payload, settings) {
    return apiRequest('/api/v1/math/forecast', {
        method: 'POST',
        body: payload,
        settings,
    });
}
export async function listModels(settings) {
    return apiRequest('/api/v1/models', { settings });
}
export async function createModel(payload, settings) {
    return apiRequest('/api/v1/models', {
        method: 'POST',
        body: payload,
        settings,
    });
}
export async function updateModel(id, payload, settings) {
    return apiRequest(`/api/v1/models/${id}`, {
        method: 'PUT',
        body: payload,
        settings,
    });
}
export async function deleteModel(id, settings) {
    return apiRequest(`/api/v1/models/${id}`, {
        method: 'DELETE',
        settings,
    });
}
