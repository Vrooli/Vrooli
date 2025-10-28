// Dynamic configuration that works in both dev and production
interface Config {
  API_URL: string;
  WS_URL: string;
  API_PORT: string;
  UI_PORT: string;
  WS_PORT: string;
}

let cachedConfig: Config | null = null;

// Get configuration dynamically - works in both dev and production
async function getConfig(): Promise<Config> {
  if (cachedConfig) {
    return cachedConfig;
  }

  // Try to get from build-time environment variables (dev mode)
  if (import.meta.env.VITE_API_URL) {
    cachedConfig = {
      API_URL: import.meta.env.VITE_API_URL,
      WS_URL: import.meta.env.VITE_WS_URL,
      API_PORT: import.meta.env.VITE_API_PORT,
      UI_PORT: import.meta.env.VITE_UI_PORT,
      WS_PORT: import.meta.env.VITE_WS_PORT,
    };
    return cachedConfig;
  }

  // Fallback to runtime config endpoint (production mode)
  const basePath = import.meta.env.BASE_URL || '/';
  const normalizedBase = basePath.endsWith('/') ? basePath : `${basePath}/`;
  const configPath = `${normalizedBase}config`;

  try {
    const response = await fetch(configPath, {
      headers: { accept: 'application/json' },
    });
    if (!response.ok) {
      throw new Error(`Config fetch failed: ${response.status}`);
    }

    const raw = await response.text();
    let data: Record<string, any>;

    try {
      data = JSON.parse(raw);
    } catch (parseError) {
      const isHtml = raw.trim().startsWith('<');
      const hint = isHtml
        ? 'Received HTML instead of JSON. Run the scenario via "make start" or ensure the production server (node server.js) is serving /config.'
        : `Response body: ${raw.slice(0, 120)}`;
      throw new Error(`Config endpoint did not return JSON. ${hint}`);
    }

    if (!data.apiUrl) {
      throw new Error('API URL not configured');
    }

    cachedConfig = {
      API_URL: data.apiUrl,
      WS_URL: data.apiUrl.replace('http', 'ws').replace('/api/v1', ''),
      API_PORT: data.apiUrl.split(':')[2]?.split('/')[0] || '24785',
      UI_PORT: window.location.port || '42969',
      WS_PORT: '29031', // Default fallback
    };
    return cachedConfig;
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error';
    throw new Error(`Failed to load configuration: ${message}`);
  }
}

// Sync version for immediate use (will throw in production without config)
function getConfigSync(): Config {
  if (cachedConfig) {
    return cachedConfig;
  }
  
  if (import.meta.env.VITE_API_URL) {
    return {
      API_URL: import.meta.env.VITE_API_URL,
      WS_URL: import.meta.env.VITE_WS_URL,
      API_PORT: import.meta.env.VITE_API_PORT,
      UI_PORT: import.meta.env.VITE_UI_PORT,
      WS_PORT: import.meta.env.VITE_WS_PORT,
    };
  }

  throw new Error('Configuration not loaded. Call getConfig() first.');
}

// Export async config getter
export { getConfig };

// Export sync getters (will throw if config not loaded)
export const config = getConfigSync;
export const getApiBase = () => config().API_URL;
export const getWsBase = () => config().WS_URL;

// Legacy exports - these will work in dev mode, fail in production until config is loaded
export const API_BASE = (() => {
  try {
    return getApiBase();
  } catch {
    return ''; // Return empty string if not available yet
  }
})();

export const WS_BASE = (() => {
  try {
    return getWsBase();
  } catch {
    return '';
  }
})();
