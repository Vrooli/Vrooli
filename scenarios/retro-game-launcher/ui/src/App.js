import React, { useState, useEffect, useCallback, useMemo } from 'react';
import axios from 'axios';
import {
  Search,
  Play,
  Star,
  Gamepad2,
  Sparkles,
  Trophy,
  Eye,
  Rocket,
  Cpu,
  Terminal,
  Code,
  Brain
} from 'lucide-react';

const ENGINE_OPTIONS = [
  { id: 'javascript', name: 'JavaScript', icon: Code, color: 'text-neon-yellow' },
  { id: 'pico8', name: 'PICO-8', icon: Gamepad2, color: 'text-neon-pink' },
  { id: 'tic80', name: 'TIC-80', icon: Terminal, color: 'text-neon-cyan' }
];

const DEFAULT_API_PORT = (process.env.REACT_APP_API_PORT && process.env.REACT_APP_API_PORT.trim()) || '8080';
const LOOPBACK_HOST = ['127', '0', '0', '1'].join('.');
const DOUBLE_SLASH = '//';
const PLAYER_MESSAGE_SOURCE = 'retro-game-launcher-player';

const initialPlayOverlayState = {
  visible: false,
  loading: false,
  gameId: null,
  game: null,
  error: null,
  runtimeError: null
};

const createInitialPlayOverlayState = () => ({ ...initialPlayOverlayState });

const normalizeBase = (value = '') => value.replace(/\/+$/, '');
const ensureLeadingSlash = (value = '') => (value.startsWith('/') ? value : `/${value}`);
const PROXY_PATH_PATTERN = /(\/apps\/[^/]+\/proxy)(?:\/|$)/i;

const isLocalHostname = (hostname) => {
  if (!hostname) return false;
  const normalized = hostname.toLowerCase();
  return ['localhost', '127.0.0.1', '0.0.0.0', '::1', '[::1]'].includes(normalized);
};

const pickProxyCandidate = (info) => {
  if (!info) return undefined;

  const primary = info.primary || info.endpoint || info.target;
  if (primary && typeof primary === 'object') {
    return primary.url || primary.path || primary.target;
  }

  if (typeof info.primary === 'string') {
    return info.primary;
  }

  if (Array.isArray(info.endpoints)) {
    for (const endpoint of info.endpoints) {
      if (!endpoint) continue;
      if (typeof endpoint === 'string') {
        return endpoint;
      }
      if (typeof endpoint === 'object') {
        const candidate = endpoint.url || endpoint.path || endpoint.target;
        if (candidate) {
          return candidate;
        }
      }
    }
  }

  if (typeof info === 'string') {
    return info;
  }

  return undefined;
};

const resolveProxyBase = () => {
  if (typeof window === 'undefined') {
    return undefined;
  }

  const proxyInfo = window.__APP_MONITOR_PROXY_INFO__ || window.__APP_MONITOR_PROXY_INDEX__;
  const candidate = pickProxyCandidate(proxyInfo);

  if (!candidate) {
    return undefined;
  }

  if (/^https?:\/\//i.test(candidate)) {
    return normalizeBase(candidate);
  }

  if (window.location && window.location.origin) {
    return normalizeBase(`${normalizeBase(window.location.origin)}${ensureLeadingSlash(candidate)}`);
  }

  return undefined;
};

const resolveLocationProxyBase = () => {
  if (typeof window === 'undefined') {
    return undefined;
  }

  const { origin, pathname } = window.location || {};
  if (!origin || !pathname) {
    return undefined;
  }

  const match = pathname.match(PROXY_PATH_PATTERN);
  if (!match || !match[1]) {
    return undefined;
  }

  const basePath = ensureLeadingSlash(match[1]);
  return normalizeBase(`${normalizeBase(origin)}${basePath}`);
};

const resolveRelativeToOrigin = (value = '') => {
  if (!value) {
    return '';
  }

  if (/^https?:\/\//i.test(value)) {
    return normalizeBase(value);
  }

  if (typeof window === 'undefined' || !window.location || !window.location.origin) {
    return '';
  }

  return normalizeBase(`${normalizeBase(window.location.origin)}${ensureLeadingSlash(value)}`);
};

const buildLocalApiOrigin = (hostname, protocol) => {
  if (!hostname) {
    return undefined;
  }

  const safeProtocol = protocol || 'http:';
  return `${safeProtocol}${DOUBLE_SLASH}${hostname}:${DEFAULT_API_PORT}`;
};

const buildLoopbackBase = () => `http:${DOUBLE_SLASH}${LOOPBACK_HOST}:${DEFAULT_API_PORT}`;

const resolveApiBaseUrl = () => {
  const explicit = (process.env.REACT_APP_API_URL || '').trim();
  if (explicit) {
    const resolved = resolveRelativeToOrigin(explicit);
    if (resolved) {
      return resolved;
    }
  }

  if (typeof window !== 'undefined') {
    const proxyBase = resolveProxyBase();
    if (proxyBase) {
      return proxyBase;
    }

    const locationProxyBase = resolveLocationProxyBase();
    if (locationProxyBase) {
      return locationProxyBase;
    }

    const { hostname, origin, protocol } = window.location || {};
    const nodeEnv = (process.env.NODE_ENV || '').toLowerCase();
    const preferCurrentOrigin = Boolean(origin) && (nodeEnv === 'production' || (hostname && !isLocalHostname(hostname)));

    if (preferCurrentOrigin) {
      return normalizeBase(origin);
    }

    if (hostname && isLocalHostname(hostname)) {
      const localCandidate = buildLocalApiOrigin(hostname, protocol);
      if (localCandidate) {
        return normalizeBase(localCandidate);
      }
    }

    if (origin) {
      return normalizeBase(origin);
    }
  }

  return normalizeBase(buildLoopbackBase());
};

const buildApiUrl = (path) => {
  const base = resolveApiBaseUrl();
  const rawPath = typeof path === 'string' ? path.trim() : '';
  const normalizedPath = rawPath.startsWith('/') ? rawPath : `/${rawPath}`;

  if (!base || !base.trim()) {
    return normalizedPath;
  }

  const normalizedBase = normalizeBase(base);

  if (/\/apps\/[^/]+\/proxy(?:$|\/)/i.test(`${normalizedBase}/`)) {
    return `${normalizedBase}${normalizedPath}`;
  }

  if (/^https?:\/\//i.test(normalizedBase)) {
    const baseUrl = normalizedBase.endsWith('/') ? normalizedBase : `${normalizedBase}/`;
    return new URL(normalizedPath, baseUrl).toString();
  }

  return `${normalizedBase}${normalizedPath}`;
};

const formatNumber = (value) => {
  try {
    return new Intl.NumberFormat('en-US').format(Number.isFinite(value) ? value : Number(value) || 0);
  } catch (error) {
    console.warn('Failed to format number', error);
    return String(value ?? '0');
  }
};

const escapeHtml = (value = '') =>
  String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');

const buildGameDocument = (game) => {
  if (!game) {
    return '';
  }

  const title = escapeHtml(game.title || 'Retro Game');
  const gameCode = typeof game.code === 'string' ? game.code : '';
  const serializedCode = JSON.stringify(gameCode);
  const serializedId = JSON.stringify(game.id || '');
  const serializedSource = JSON.stringify(PLAYER_MESSAGE_SOURCE);

  return `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width,initial-scale=1" />
    <title>${title}</title>
    <style>
      :root { color-scheme: dark; }
      * { box-sizing: border-box; }
      html, body {
        margin: 0;
        padding: 0;
        width: 100%;
        height: 100%;
        background: #05010f;
        color: #00f0ff;
        font-family: 'Fira Code', monospace;
      }
      body {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: flex-start;
      }
      #retro-player-container {
        width: 100%;
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 16px;
      }
      #retro-player-error {
        max-width: min(720px, 92%);
        border: 1px solid rgba(255, 64, 160, 0.55);
        background: rgba(255, 0, 128, 0.12);
        color: #ff4da6;
        padding: 18px;
        border-radius: 16px;
        box-shadow: 0 0 35px rgba(255, 0, 170, 0.28);
        line-height: 1.6;
      }
      #retro-player-error h2 {
        margin: 0 0 12px 0;
        font-size: 16px;
        text-transform: uppercase;
        letter-spacing: 0.12em;
      }
      #retro-player-error pre {
        margin-top: 12px;
        max-height: 45vh;
        overflow: auto;
        white-space: pre-wrap;
        word-break: break-word;
        font-size: 12px;
        background: rgba(5, 5, 20, 0.85);
        padding: 12px;
        border-radius: 12px;
        border: 1px solid rgba(0, 255, 255, 0.2);
        color: #e5e7ff;
      }
      canvas {
        image-rendering: pixelated;
      }
    </style>
  </head>
  <body>
    <div id="retro-player-container"></div>
    <script>
      const GAME_ID = ${serializedId};
      const PLAYER_SOURCE = ${serializedSource};

      function ensureContainer() {
        let container = document.getElementById('retro-player-container');
        if (!container) {
          container = document.createElement('div');
          container.id = 'retro-player-container';
          container.style.padding = '16px';
          container.style.display = 'flex';
          container.style.alignItems = 'center';
          container.style.justifyContent = 'center';
          document.body.appendChild(container);
        }
        return container;
      }

      function postToParent(type, payload) {
        try {
          parent.postMessage(Object.assign({ type, gameId: GAME_ID, source: PLAYER_SOURCE }, payload || {}), '*');
        } catch (error) {
          console.warn('[RetroGameLauncher] Unable to notify parent', error);
        }
      }

      function normalizeError(error) {
        if (!error) {
          return { message: 'Unknown error' };
        }
        if (typeof error === 'string') {
          return { message: error };
        }
        return {
          message: error.message || 'Unknown error',
          stack: error.stack || undefined
        };
      }

      function renderRuntimeError(error) {
        const normalized = normalizeError(error);
        const container = ensureContainer();
        container.innerHTML = '';
        const panel = document.createElement('div');
        panel.id = 'retro-player-error';

        const title = document.createElement('h2');
        title.textContent = 'Game Error';
        panel.appendChild(title);

        const message = document.createElement('div');
        message.textContent = normalized.message;
        panel.appendChild(message);

        if (normalized.stack) {
          const stack = document.createElement('pre');
          stack.textContent = normalized.stack;
          panel.appendChild(stack);
        }

        container.appendChild(panel);
      }

      window.addEventListener('error', function (event) {
        const normalized = normalizeError(event && (event.error || event.message));
        postToParent('runtime-error', normalized);
        renderRuntimeError(normalized);
      });

      window.addEventListener('unhandledrejection', function (event) {
        const normalized = normalizeError(event && event.reason);
        postToParent('runtime-error', normalized);
        renderRuntimeError(normalized);
      });

      (function runGame() {
        try {
          const GAME_CODE = ${serializedCode};
          const executor = new Function(GAME_CODE);
          executor();
          postToParent('runtime-ready');
        } catch (error) {
          const normalized = normalizeError(error);
          postToParent('runtime-error', normalized);
          renderRuntimeError(normalized);
        }
      })();
    </script>
  </body>
</html>`;
};

function App() {
  const [games, setGames] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedGame, setSelectedGame] = useState(null);
  const [generatingGame, setGeneratingGame] = useState(false);
  const [gamePrompt, setGamePrompt] = useState('');
  const [selectedEngine, setSelectedEngine] = useState('javascript');
  const [activeTab, setActiveTab] = useState('browse');
  const [featuredGames, setFeaturedGames] = useState([]);
  const [error, setError] = useState(null);
  const [notification, setNotification] = useState(null);
  const [playOverlay, setPlayOverlay] = useState(createInitialPlayOverlayState());

  const engines = ENGINE_OPTIONS;

  const resetPlayOverlay = useCallback(() => {
    setPlayOverlay(createInitialPlayOverlayState());
  }, []);

  const updatePlayOverlay = useCallback((updates) => {
    setPlayOverlay((previous) => ({ ...previous, ...updates }));
  }, []);

  const promptTemplates = [
    'Create a simple platformer with a blue character who can jump and collect coins',
    'Make a space shooter where you dodge asteroids and shoot enemies',
    'Build a puzzle game with colored blocks that disappear when matched',
    'Create a retro snake game with neon graphics and power-ups',
    'Make a racing game viewed from above with multiple cars',
    'Build a breakout clone with special power-up bricks',
    'Create a maze game where you collect all dots while avoiding ghosts',
    'Make a tower defense game with robots attacking a base'
  ];

  const totalPlays = useMemo(
    () => games.reduce((sum, game) => sum + (Number(game.play_count) || 0), 0),
    [games]
  );

  const heroStats = useMemo(() => {
    const engineSet = new Set();
    games.forEach((game) => {
      if (game.engine) {
        engineSet.add(game.engine);
      }
    });

    return [
      { id: 'games', label: 'Playable Games', value: formatNumber(games.length || 0) },
      { id: 'featured', label: 'Featured Launches', value: formatNumber(featuredGames.length || 0) },
      { id: 'plays', label: 'Total Plays Logged', value: formatNumber(totalPlays) },
      { id: 'engines', label: 'Engines Supported', value: formatNumber(Math.max(engineSet.size, engines.length)) }
    ];
  }, [games, featuredGames, totalPlays, engines.length]);

  const popularTags = useMemo(() => {
    const tagCounts = new Map();
    games.forEach((game) => {
      (game.tags || []).forEach((tag) => {
        if (!tag) return;
        const normalized = String(tag).trim();
        if (!normalized) return;
        tagCounts.set(normalized, (tagCounts.get(normalized) || 0) + 1);
      });
    });

    return Array.from(tagCounts.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 4)
      .map(([tag]) => tag);
  }, [games]);

  const fetchGames = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await axios.get(buildApiUrl('/api/games'));
      setGames(response.data || []);
    } catch (err) {
      setError('Failed to load games. Make sure the API is running.');
      console.error('Error fetching games:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchFeaturedGames = useCallback(async () => {
    try {
      const response = await axios.get(buildApiUrl('/api/featured'));
      setFeaturedGames(response.data || []);
    } catch (err) {
      console.error('Error fetching featured games:', err);
    }
  }, []);

  const searchGames = useCallback(async (query) => {
    if (!query.trim()) {
      fetchGames();
      return;
    }

    try {
      setLoading(true);
      const response = await axios.get(buildApiUrl('/api/search/games'), { params: { q: query } });
      setGames(response.data || []);
    } catch (err) {
      setError('Search failed. Please try again.');
      console.error('Error searching games:', err);
    } finally {
      setLoading(false);
    }
  }, [fetchGames]);

  const generateGame = async () => {
    if (!gamePrompt.trim()) return;

    try {
      setGeneratingGame(true);
      setError(null);

      const response = await axios.post(buildApiUrl('/api/generate'), {
        prompt: gamePrompt,
        engine: selectedEngine,
        tags: ['ai-generated', 'retro']
      });

      setNotification(`Game generation started! Generation ID: ${response.data.generation_id}`);
      setGamePrompt('');
      fetchGames();
    } catch (err) {
      setError('Failed to start game generation. Please try again.');
      console.error('Error generating game:', err);
    } finally {
      setGeneratingGame(false);
    }
  };

  const playGame = useCallback(
    async (gameInput) => {
      const gameId = typeof gameInput === 'string' ? gameInput : gameInput?.id;
      if (!gameId) {
        return;
      }

      const knownGame =
        typeof gameInput === 'object' && gameInput
          ? gameInput
          : games.find((entry) => entry.id === gameId);

      updatePlayOverlay({
        visible: true,
        loading: true,
        gameId,
        game: knownGame && typeof knownGame.code === 'string' ? knownGame : null,
        error: null,
        runtimeError: null
      });

      try {
        await axios.post(buildApiUrl(`/api/games/${gameId}/play`), {});

        setGames((currentGames) =>
          currentGames.map((game) =>
            game.id === gameId
              ? { ...game, play_count: (Number(game.play_count) || 0) + 1 }
              : game
          )
        );

        let gameToLaunch = knownGame;
        if (!gameToLaunch || typeof gameToLaunch.code !== 'string') {
          const response = await axios.get(buildApiUrl(`/api/games/${gameId}`));
          gameToLaunch = response.data;
        }

        updatePlayOverlay({
          loading: false,
          error: null,
          runtimeError: null,
          game: gameToLaunch || null
        });
      } catch (err) {
        console.error('Error launching game:', err);
        const message =
          err?.response?.data?.error ||
          err?.response?.data?.message ||
          err?.message ||
          'Unable to launch game. Please try again.';

        updatePlayOverlay({
          loading: false,
          error: message || 'Unable to launch game. Please try again.'
        });
      }
    },
    [games, updatePlayOverlay]
  );

  useEffect(() => {
    fetchGames();
    fetchFeaturedGames();
  }, [fetchGames, fetchFeaturedGames]);

  useEffect(() => {
    const timeoutId = setTimeout(() => {
      searchGames(searchTerm);
    }, 300);

    return () => clearTimeout(timeoutId);
  }, [searchTerm, searchGames]);

  useEffect(() => {
    if (!notification) return;
    const timeoutId = setTimeout(() => setNotification(null), 4000);
    return () => clearTimeout(timeoutId);
  }, [notification]);

  useEffect(() => {
    if (!playOverlay.visible) {
      return undefined;
    }

    const handleMessage = (event) => {
      if (!event?.data || typeof event.data !== 'object') {
        return;
      }

      const { source, type, message: runtimeMessage, stack, gameId } = event.data;
      if (source !== PLAYER_MESSAGE_SOURCE) {
        return;
      }

      if (gameId && playOverlay.gameId && gameId !== playOverlay.gameId) {
        return;
      }

      if (type === 'runtime-error') {
        updatePlayOverlay({
          runtimeError: {
            message: runtimeMessage || 'The game encountered an unexpected error.',
            stack: stack || null
          }
        });
      }

      if (type === 'runtime-ready') {
        updatePlayOverlay({ runtimeError: null });
      }
    };

    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, [playOverlay.visible, playOverlay.gameId, updatePlayOverlay]);

  const LoadingSpinner = () => (
    <div className="flex justify-center items-center py-8">
      <div className="loading-spinner"></div>
      <span className="ml-3 text-neon-cyan font-mono">Loading games...</span>
    </div>
  );

  const ErrorMessage = ({ message }) => (
    <div className="notification-banner notification-banner--error">
      <span>{message}</span>
      <button onClick={() => setError(null)} className="notification-banner__close">Dismiss</button>
    </div>
  );

  const NotificationBanner = ({ message }) => (
    <div className="notification-banner">
      <Sparkles className="w-4 h-4" />
      <span>{message}</span>
      <button onClick={() => setNotification(null)} className="notification-banner__close">Dismiss</button>
    </div>
  );

  const GameCard = ({ game, featured = false }) => {
    const engineMeta = engines.find((engine) => engine.id === game.engine);
    const createdAt = game.created_at ? new Date(game.created_at) : null;

    return (
      <article
        className={`game-card ${featured ? 'game-card--featured' : ''}`}
        onClick={() => setSelectedGame(game)}
      >
        <div className="game-card__header">
          <h3 className="game-card__title">{game.title}</h3>
          {featured && (
            <span className="game-card__badge">
              <Star className="w-3 h-3" /> Featured
            </span>
          )}
        </div>

        <p className="game-card__description">
          {game.description || 'No description available'}
        </p>

        <div className="game-card__meta">
          <span className={`game-card__engine ${engineMeta ? engineMeta.color : 'text-neon-cyan'}`}>
            {game.engine}
          </span>
          <div className="game-card__stats">
            <span className="game-card__stat">
              <Eye className="w-3 h-3" />
              {(Number(game.play_count) || 0).toLocaleString()}
            </span>
            {createdAt && (
              <span className="game-card__stat">
                <span>{createdAt.toLocaleDateString()}</span>
              </span>
            )}
          </div>
        </div>

        {game.tags && game.tags.length > 0 && (
          <div className="game-card__tags">
            {game.tags.slice(0, 3).map((tag, index) => (
              <span key={index} className="game-card__tag">#{tag}</span>
            ))}
          </div>
        )}

        <button
          className="neon-button game-card__cta"
          onClick={(event) => {
            event.stopPropagation();
            playGame(game);
          }}
        >
          <Play className="w-4 h-4 mr-2" /> Play Game
        </button>
      </article>
    );
  };

  const GameModal = ({ game, onClose }) => {
    if (!game) return null;
    const engineMeta = engines.find((engine) => engine.id === game.engine);

    return (
      <div className="modal-overlay">
        <div className="modal-card">
          <div className="modal-card__header">
            <div>
              <span className="brand-pill modal-card__pill">
                <Rocket className="w-3 h-3 mr-1" /> Retro Game Launcher
              </span>
              <h2 className="modal-card__title">{game.title}</h2>
            </div>
            <button onClick={onClose} className="modal-card__close">×</button>
          </div>

          <div className="modal-card__body">
            <div className="modal-card__summary">
              <div>
                <span className="modal-card__label">Engine</span>
                <span className={`modal-card__value ${engineMeta ? engineMeta.color : 'text-neon-cyan'}`}>
                  {game.engine}
                </span>
              </div>
              <div>
                <span className="modal-card__label">Plays</span>
                <span className="modal-card__value">{(Number(game.play_count) || 0).toLocaleString()}</span>
              </div>
              {game.rating && (
                <div>
                  <span className="modal-card__label">Rating</span>
                  <span className="modal-card__value">{Number(game.rating).toFixed(1)} ★</span>
                </div>
              )}
              {game.created_at && (
                <div>
                  <span className="modal-card__label">Created</span>
                  <span className="modal-card__value">{new Date(game.created_at).toLocaleString()}</span>
                </div>
              )}
            </div>

            {game.description && (
              <section className="modal-card__section">
                <h3 className="modal-card__section-title">Description</h3>
                <p className="modal-card__text">{game.description}</p>
              </section>
            )}

            {game.prompt && (
              <section className="modal-card__section">
                <h3 className="modal-card__section-title">Original Prompt</h3>
                <p className="modal-card__prompt">"{game.prompt}"</p>
              </section>
            )}

            {game.tags && game.tags.length > 0 && (
              <section className="modal-card__section">
                <h3 className="modal-card__section-title">Tags</h3>
                <div className="modal-card__tags">
                  {game.tags.map((tag, index) => (
                    <span key={index} className="modal-card__tag">#{tag}</span>
                  ))}
                </div>
              </section>
            )}
          </div>

          <div className="modal-card__footer">
            <button
              className="neon-button hero-primary"
              onClick={() => playGame(game)}
            >
              <Play className="w-4 h-4 mr-2" /> Jump In
            </button>
            <button className="hero-secondary" onClick={onClose}>
              Close
            </button>
          </div>
        </div>
      </div>
    );
  };

  const GamePlayerOverlay = ({ state, onClose, onRetry }) => {
    if (!state.visible) {
      return null;
    }

    const { game, loading, error: launchError, runtimeError } = state;
    const title = game?.title || 'Retro Game';
    const canRetry = typeof onRetry === 'function' && Boolean(state.gameId);

    return (
      <div className="play-overlay" role="dialog" aria-modal="true" aria-label={`Playing ${title}`}>
        <div className="play-overlay__backdrop" onClick={onClose} />
        <div className="play-overlay__panel">
          <div className="play-overlay__header">
            <div>
              <span className="brand-pill play-overlay__brand">
                <Play className="w-4 h-4 mr-2" /> Launch Bay
              </span>
              <h2 className="play-overlay__title">{title}</h2>
              {game?.engine && (
                <p className="play-overlay__subtitle">Engine: {game.engine}</p>
              )}
            </div>
            <button onClick={onClose} className="play-overlay__close" aria-label="Close player">
              ×
            </button>
          </div>

          <div className="play-overlay__body">
            {loading && (
              <div className="play-overlay__status">
                <div className="loading-spinner"></div>
                <p className="play-overlay__status-text">Launching game...</p>
              </div>
            )}

            {!loading && launchError && (
              <div className="play-overlay__status play-overlay__status--error">
                <p className="play-overlay__status-text">{launchError}</p>
                {canRetry && (
                  <button className="neon-button play-overlay__retry" onClick={onRetry}>
                    Try Again
                  </button>
                )}
              </div>
            )}

            {!loading && !launchError && game && (
              <div className="play-overlay__frame-wrapper">
                <iframe
                  title={`${title} player`}
                  className="play-overlay__frame"
                  sandbox="allow-scripts allow-pointer-lock"
                  srcDoc={buildGameDocument(game)}
                />

                {runtimeError && (
                  <div className="play-overlay__status play-overlay__status--error play-overlay__status--runtime">
                    <p className="play-overlay__status-text">{runtimeError.message}</p>
                    {runtimeError.stack && (
                      <pre className="play-overlay__stack">{runtimeError.stack}</pre>
                    )}
                    {canRetry && (
                      <button className="neon-button play-overlay__retry" onClick={onRetry}>
                        Reload Game
                      </button>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="app-shell">
      <div className="app-shell__decor">
        <div className="app-shell__grid" />
        <div className="app-shell__glow app-shell__glow--top" />
        <div className="app-shell__glow app-shell__glow--bottom" />
      </div>

      <div className="app-shell__content">
        <header className="app-header glass-panel">
          <div className="app-header__row">
            <div className="app-brand">
              <span className="brand-pill">
                <Rocket className="w-4 h-4 mr-2" /> Retro Game Launcher
              </span>
              <h1 className="hero-title">Launch Into Neon Nights</h1>
              <p className="hero-subtitle">
                Curate pixel-perfect classics, spin up AI-powered remixes, and keep your arcade alive forever.
              </p>
            </div>
            <div className="hero-actions">
              <button
                className="neon-button hero-primary"
                onClick={() => setActiveTab('generate')}
              >
                <Sparkles className="w-4 h-4 mr-2" />
                Start a New Game
              </button>
              <button
                className="hero-secondary"
                onClick={() => setActiveTab('browse')}
              >
                <Gamepad2 className="w-4 h-4 mr-2" />
                Browse Library
              </button>
            </div>
          </div>

          <div className="app-header__stats">
            {heroStats.map((stat) => (
              <div key={stat.id} className="stat-card">
                <span className="stat-card__value">{stat.value}</span>
                <span className="stat-card__label">{stat.label}</span>
              </div>
            ))}
          </div>

          {popularTags.length > 0 && (
            <div className="app-header__tags">
              <span className="tag-label">Popular tags:</span>
              <div className="tag-list">
                {popularTags.map((tag) => (
                  <span key={tag} className="tag-chip">#{tag}</span>
                ))}
              </div>
            </div>
          )}
        </header>

        <nav className="app-nav glass-panel">
          <div className="app-nav__tabs">
            {[
              { id: 'browse', label: 'Browse Games', icon: Gamepad2 },
              { id: 'generate', label: 'Generate', icon: Brain },
              { id: 'featured', label: 'Featured', icon: Star }
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`tab-pill ${activeTab === tab.id ? 'tab-pill--active' : ''}`}
              >
                <tab.icon className="w-4 h-4 mr-2" />
                {tab.label}
              </button>
            ))}
          </div>
          <div className="app-nav__status">
            <Cpu className="w-4 h-4 mr-2" />
            <span>Systems nominal</span>
          </div>
        </nav>

        <main className="app-main space-y-8">
          {notification && <NotificationBanner message={notification} />}
          {error && <ErrorMessage message={error} />}

          {activeTab === 'browse' && (
            <section className="glass-panel">
              <div className="panel-header">
                <div>
                  <h2 className="panel-title">Explore the Library</h2>
                  <p className="panel-subtitle">
                    Discover neon-coded adventures curated by the launcher core.
                  </p>
                </div>
                <button
                  className="panel-link"
                  onClick={() => setActiveTab('generate')}
                >
                  <Sparkles className="w-4 h-4 mr-2" /> Generate with AI
                </button>
              </div>

              <div className="search-stack">
                <div className="search-field">
                  <Search className="search-field__icon" />
                  <input
                    type="text"
                    placeholder="Search retro games..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="search-input w-full pl-10"
                  />
                </div>
              </div>

              {loading ? (
                <LoadingSpinner />
              ) : games.length > 0 ? (
                <div className="game-grid">
                  {games.map((game) => (
                    <GameCard key={game.id} game={game} />
                  ))}
                </div>
              ) : (
                <div className="empty-state">
                  <Gamepad2 className="empty-state__icon" />
                  <h3 className="empty-state__title">No games matched that search</h3>
                  <p className="empty-state__copy">
                    Try a different query or generate something brand new.
                  </p>
                </div>
              )}
            </section>
          )}

          {activeTab === 'generate' && (
            <section className="glass-panel">
              <div className="panel-header">
                <div>
                  <h2 className="panel-title">Build a New Release</h2>
                  <p className="panel-subtitle">
                    Pick an engine, describe the vibe, and let the synthesizer spin up a prototype.
                  </p>
                </div>
              </div>

              <div className="grid gap-8 lg:grid-cols-3">
                <div className="lg:col-span-2 space-y-6">
                  <div>
                    <label className="panel-label">Game Engine</label>
                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                      {engines.map((engine) => (
                        <button
                          key={engine.id}
                          onClick={() => setSelectedEngine(engine.id)}
                          className={`engine-pill ${selectedEngine === engine.id ? 'engine-pill--active' : ''}`}
                        >
                          <engine.icon className="w-4 h-4 mr-2" />
                          {engine.name}
                        </button>
                      ))}
                    </div>
                  </div>

                  <div>
                    <label className="panel-label">Game Description</label>
                    <textarea
                      value={gamePrompt}
                      onChange={(e) => setGamePrompt(e.target.value)}
                      placeholder="Describe the game you want to create..."
                      className="prompt-area"
                    />
                  </div>

                  <button
                    onClick={generateGame}
                    disabled={!gamePrompt.trim() || generatingGame}
                    className="neon-button hero-primary w-full py-4 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {generatingGame ? (
                      <>
                        <div className="loading-spinner inline mr-2"></div>
                        Generating Game...
                      </>
                    ) : (
                      <>
                        <Sparkles className="w-5 h-5 inline mr-2" />
                        Generate Game
                      </>
                    )}
                  </button>
                </div>

                <div className="space-y-4">
                  <div className="panel-label">Try These Ideas</div>
                  <div className="idea-stack">
                    {promptTemplates.slice(0, 6).map((template, index) => (
                      <button
                        key={index}
                        onClick={() => setGamePrompt(template)}
                        className="idea-chip"
                      >
                        {template}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            </section>
          )}

          {activeTab === 'featured' && (
            <section className="glass-panel">
              <div className="panel-header">
                <div>
                  <h2 className="panel-title">Featured Showcase</h2>
                  <p className="panel-subtitle">
                    Hand-picked gems from the community and core maintainers.
                  </p>
                </div>
              </div>

              {featuredGames.length > 0 ? (
                <div className="game-grid">
                  {featuredGames.map((game) => (
                    <GameCard key={game.id} game={game} featured={true} />
                  ))}
                </div>
              ) : (
                <div className="empty-state">
                  <Trophy className="empty-state__icon" />
                  <h3 className="empty-state__title">No featured games yet</h3>
                  <p className="empty-state__copy">
                    Generate something legendary and make the highlights reel.
                  </p>
                </div>
              )}
            </section>
          )}
        </main>

        <footer className="app-footer glass-panel">
          <div className="app-footer__content">
            <span className="brand-pill">
              <Rocket className="w-4 h-4 mr-2" /> Retro Game Launcher
            </span>
            <p className="app-footer__copy">Powered by AI • Built for synthwave storytellers.</p>
          </div>
        </footer>
      </div>

      {selectedGame && (
        <GameModal game={selectedGame} onClose={() => setSelectedGame(null)} />
      )}
      <GamePlayerOverlay
        state={playOverlay}
        onClose={resetPlayOverlay}
        onRetry={() => {
          if (!playOverlay.gameId) return;
          const fallbackGame = playOverlay.game || games.find((entry) => entry.id === playOverlay.gameId);
          playGame(fallbackGame || playOverlay.gameId);
        }}
      />
    </div>
  );
}

export default App;
