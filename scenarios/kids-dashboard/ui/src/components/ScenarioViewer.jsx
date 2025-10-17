import React, { useState, useEffect, useMemo } from 'react';
import { motion } from 'framer-motion';
import { ArrowLeft, Home, RefreshCw, Maximize2, Minimize2 } from 'lucide-react';

const getWindowChain = () => {
  if (typeof window === 'undefined') {
    return [];
  }

  const chain = [];
  const seen = new Set();

  const pushUnique = (candidate) => {
    if (!candidate || typeof candidate !== 'object' || seen.has(candidate)) {
      return;
    }
    seen.add(candidate);
    chain.push(candidate);
  };

  pushUnique(window);

  try {
    if (window.parent && window.parent !== window) {
      pushUnique(window.parent);
    }
  } catch (_) {
    // Accessing parent can throw in cross-origin contexts; ignore.
  }

  try {
    if (window.top && window.top !== window && window.top !== window.parent) {
      pushUnique(window.top);
    }
  } catch (_) {
    // Ignore inaccessible top window.
  }

  return chain;
};

const collectProxyEntries = () => {
  const candidates = [];
  const seenKeys = new Set();

  const pushEntry = (entry, owner) => {
    if (!entry || typeof entry !== 'object') {
      return;
    }

    const keyParts = [
      entry.url,
      entry.target,
      entry.path,
      entry.slug,
      entry.label,
      entry.normalizedLabel,
      entry.port,
    ].filter(Boolean);

    if (keyParts.length === 0) {
      return;
    }

    const key = keyParts.join('|');
    if (seenKeys.has(key)) {
      return;
    }

    seenKeys.add(key);
    candidates.push({ entry, owner });
  };

  const windows = getWindowChain();

  const safeRead = (fn) => {
    try {
      return fn();
    } catch (_) {
      return undefined;
    }
  };

  windows.forEach((win) => {
    const info = safeRead(() => win.__APP_MONITOR_PROXY_INFO__);
    if (info?.primary) {
      pushEntry(info.primary, win);
    }
    if (Array.isArray(info?.ports)) {
      info.ports.forEach((entry) => pushEntry(entry, win));
    }

    const index = safeRead(() => win.__APP_MONITOR_PROXY_INDEX__);
    if (index) {
      if (index.aliasMap instanceof Map) {
        index.aliasMap.forEach((value) => pushEntry(value, win));
      } else if (Array.isArray(index)) {
        index.forEach((value) => pushEntry(value, win));
      } else if (typeof index === 'object') {
        Object.values(index).forEach((value) => pushEntry(value, win));
      }
    }
  });

  return candidates;
};

const resolveProxyEntryUrl = (entry, owner) => {
  const candidatePath = entry?.url || entry?.path || entry?.target;
  if (!candidatePath) {
    return null;
  }

  if (typeof candidatePath === 'string' && /^https?:\/\//i.test(candidatePath)) {
    return candidatePath;
  }

  const ownerOrigin = owner?.location?.origin || (typeof window !== 'undefined' ? window.location?.origin : '');
  if (!ownerOrigin) {
    return null;
  }

  try {
    const url = new URL(candidatePath, ownerOrigin);
    return url.toString();
  } catch (error) {
    console.warn('[ScenarioViewer] Failed to normalize proxy entry', error);
    return null;
  }
};

const getLocationCandidates = () => {
  if (typeof window === 'undefined') {
    return [];
  }

  const locations = [];
  getWindowChain().forEach((win) => {
    try {
      if (win.location) {
        locations.push(win.location);
      }
    } catch (_) {
      // Ignore cross-origin access.
    }
  });

  return locations;
};

const deriveProxyUrlFromLocations = (slug) => {
  if (!slug) {
    return null;
  }

  const locations = getLocationCandidates();

  for (const location of locations) {
    const origin = location?.origin;
    const pathname = location?.pathname;

    const isProxyPath = typeof pathname === 'string'
      && pathname.includes('/apps/')
      && pathname.includes('/proxy/');

    if (origin && isProxyPath) {
      const encodedSlug = encodeURIComponent(slug);
      return `${origin.replace(/\/$/, '')}/apps/${encodedSlug}/proxy/`;
    }
  }

  return null;
};

const collectSlugCandidates = (scenario) => {
  return [
    scenario?.proxySlug,
    scenario?.slug,
    scenario?.name,
    scenario?.id,
  ]
    .filter(Boolean)
    .map((value) => String(value).trim().toLowerCase());
};

const resolveProxyUrl = (scenario) => {
  if (!scenario) {
    return null;
  }

  const slugCandidates = collectSlugCandidates(scenario);
  if (slugCandidates.length === 0) {
    return null;
  }

  const portCandidate = typeof scenario.port === 'number'
    ? scenario.port
    : parseInt(scenario.port, 10);

  const entries = collectProxyEntries();

  for (const { entry, owner } of entries) {
    const aliases = new Set();

    if (entry.slug) aliases.add(String(entry.slug).trim().toLowerCase());
    if (entry.label) aliases.add(String(entry.label).trim().toLowerCase());
    if (entry.normalizedLabel) aliases.add(String(entry.normalizedLabel).trim().toLowerCase());
    if (Array.isArray(entry.aliases)) {
      entry.aliases.forEach((alias) => alias && aliases.add(String(alias).trim().toLowerCase()));
    }
    if (typeof entry.port === 'number') {
      aliases.add(String(entry.port));
    }

    const matchedAlias = slugCandidates.some((candidate) => aliases.has(candidate));
    const portMatches = Number.isFinite(portCandidate) && aliases.has(String(portCandidate));

    if (!matchedAlias && !portMatches) {
      continue;
    }

    const resolvedUrl = resolveProxyEntryUrl(entry, owner);
    if (resolvedUrl) {
      return resolvedUrl;
    }
  }

  return deriveProxyUrlFromLocations(slugCandidates[0]);
};

const resolveScenarioUrl = (scenario) => {
  if (!scenario) {
    return '';
  }

  const directUrlCandidates = [
    scenario.launchUrl,
    scenario.url,
    scenario.proxyUrl,
    scenario.playUrl,
    scenario.proxyPath,
  ].filter(Boolean);

  if (directUrlCandidates.length > 0) {
    return directUrlCandidates[0];
  }

  const proxiedUrl = resolveProxyUrl(scenario);
  if (proxiedUrl) {
    return proxiedUrl;
  }

  const slugCandidates = collectSlugCandidates(scenario);

  if (scenario?.port) {
    const portValue = parseInt(scenario.port, 10);
    const port = Number.isFinite(portValue) ? portValue : scenario.port;
    const remoteProxyUrl = deriveProxyUrlFromLocations(slugCandidates[0]);
    if (remoteProxyUrl) {
      return remoteProxyUrl;
    }
    return `http://localhost:${port}`;
  }

  return '';
};

const ScenarioViewer = ({ scenario, onBack }) => {
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [iframeKey, setIframeKey] = useState(0);
  const [resolutionError, setResolutionError] = useState('');

  const scenarioUrl = useMemo(() => resolveScenarioUrl(scenario), [scenario]);

  useEffect(() => {
    setIsLoading(true);
    setResolutionError('');
    setIframeKey((prev) => prev + 1);

    if (!scenarioUrl) {
      setResolutionError('We could not find a safe way to launch this scenario.');
      setIsLoading(false);
    }
  }, [scenarioUrl, scenario?.id]);

  // Build the scenario URL
  const handleRefresh = () => {
    if (!scenarioUrl) {
      return;
    }
    setIsLoading(true);
    setIframeKey(prev => prev + 1);
  };

  const handleFullscreen = () => {
    if (!isFullscreen) {
      document.documentElement.requestFullscreen?.();
    } else {
      document.exitFullscreen?.();
    }
    setIsFullscreen(!isFullscreen);
  };

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
      if (document.fullscreenElement) {
        document.exitFullscreen();
      }
    };
  }, []);

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.9 }}
      className="min-h-screen flex flex-col bg-gradient-to-br from-blue-50 to-purple-50"
    >
      {/* Header Bar */}
      <motion.header
        initial={{ y: -50 }}
        animate={{ y: 0 }}
        className="bg-white shadow-lg p-4 flex items-center justify-between"
      >
        <div className="flex items-center gap-4">
          <motion.button
            whileHover={{ scale: 1.1 }}
            whileTap={{ scale: 0.9 }}
            onClick={onBack}
            className="p-3 bg-kid-red text-white rounded-full shadow-lg hover:bg-red-500 transition-colors"
            aria-label="Back to dashboard"
          >
            <ArrowLeft size={24} />
          </motion.button>
          
          <div className="flex items-center gap-3">
            <span className="text-4xl">{scenario.icon || '🎮'}</span>
            <div>
              <h2 className="text-2xl font-fredoka font-bold text-gray-800">
                {scenario.title}
              </h2>
              <p className="text-gray-600 text-sm">{scenario.description}</p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <motion.button
            whileHover={{ scale: 1.1 }}
            whileTap={{ scale: 0.9 }}
            onClick={handleRefresh}
            className="p-3 bg-kid-yellow text-gray-800 rounded-full shadow-lg hover:bg-yellow-400 transition-colors"
            aria-label="Refresh"
          >
            <RefreshCw size={20} />
          </motion.button>
          
          <motion.button
            whileHover={{ scale: 1.1 }}
            whileTap={{ scale: 0.9 }}
            onClick={handleFullscreen}
            className="p-3 bg-kid-teal text-white rounded-full shadow-lg hover:bg-teal-500 transition-colors"
            aria-label="Toggle fullscreen"
          >
            {isFullscreen ? <Minimize2 size={20} /> : <Maximize2 size={20} />}
          </motion.button>
          
          <motion.button
            whileHover={{ scale: 1.1 }}
            whileTap={{ scale: 0.9 }}
            onClick={onBack}
            className="p-3 bg-kid-green text-white rounded-full shadow-lg hover:bg-green-500 transition-colors"
            aria-label="Home"
          >
            <Home size={20} />
          </motion.button>
        </div>
      </motion.header>

      {/* Iframe Container */}
      <div className="flex-1 relative p-4">
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.2 }}
          className="w-full h-full bg-white rounded-3xl shadow-2xl overflow-hidden relative"
        >
          {isLoading && (
            <div className="absolute inset-0 flex items-center justify-center bg-white z-10">
              <motion.div
                animate={{ rotate: 360 }}
                transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                className="text-6xl"
              >
                🌀
              </motion.div>
            </div>
          )}
          {resolutionError ? (
            <div className="absolute inset-0 flex items-center justify-center bg-white text-center px-6">
              <div className="space-y-2">
                <p className="text-xl font-semibold text-gray-800">{resolutionError}</p>
                <p className="text-gray-500">Please ask a grown-up to restart the scenario or check its proxy settings.</p>
              </div>
            </div>
          ) : (
            <iframe
              key={iframeKey}
              src={scenarioUrl}
              className="w-full h-full border-0"
              title={scenario.title}
              sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
              onLoad={() => setIsLoading(false)}
              style={{ minHeight: 'calc(100vh - 200px)' }}
            />
          )}
        </motion.div>
      </div>

      {/* Fun decorative elements */}
      <motion.div
        className="absolute bottom-4 left-4 text-4xl pointer-events-none"
        animate={{ rotate: [0, 360] }}
        transition={{ duration: 20, repeat: Infinity, ease: "linear" }}
      >
        ⚙️
      </motion.div>
      
      <motion.div
        className="absolute bottom-4 right-4 text-4xl pointer-events-none"
        animate={{ y: [0, -10, 0] }}
        transition={{ duration: 2, repeat: Infinity }}
      >
        🎯
      </motion.div>
    </motion.div>
  );
};

export default ScenarioViewer;
