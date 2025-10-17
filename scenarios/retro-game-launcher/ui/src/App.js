import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import { 
  Search, 
  Play, 
  Star, 
  Gamepad2, 
  Zap, 
  Code, 
  Sparkles, 
  Trophy,
  Clock,
  Users,
  Eye,
  Rocket,
  Cpu,
  Terminal,
  Palette,
  Brain
} from 'lucide-react';

const DEFAULT_API_PORT = (process.env.REACT_APP_API_PORT && process.env.REACT_APP_API_PORT.trim()) || '8080';
const LOOPBACK_DEFAULT = `http://127.0.0.1:${DEFAULT_API_PORT}`;

const normalizeBase = (value = '') => value.replace(/\/+$/, '');
const ensureLeadingSlash = (value = '') => (value.startsWith('/') ? value : `/${value}`);

const isLocalHostname = (hostname) => {
  if (!hostname) return false;
  const normalized = hostname.toLowerCase();
  return [
    'localhost',
    '127.0.0.1',
    '0.0.0.0',
    '::1',
    '[::1]'
  ].includes(normalized);
};

const isLikelyProxiedPath = (pathname) => {
  if (!pathname) return false;
  return pathname.includes('/apps/') && pathname.includes('/proxy/');
};

const pickProxyCandidate = (info) => {
  if (!info) return undefined;

  const primary = info.primary || info.endpoint || info.target;
  if (primary && typeof primary === 'object') {
    const candidate = primary.url || primary.path || primary.target;
    if (candidate) {
      return candidate;
    }
  }

  if (info.primary && typeof info.primary === 'string') {
    return info.primary;
  }

  if (Array.isArray(info.endpoints)) {
    for (const endpoint of info.endpoints) {
      if (endpoint && typeof endpoint === 'object') {
        const candidate = endpoint.url || endpoint.path || endpoint.target;
        if (candidate) {
          return candidate;
        }
      } else if (typeof endpoint === 'string' && endpoint.trim()) {
        return endpoint;
      }
    }
  }

  if (typeof info === 'string' && info.trim()) {
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

  const origin = window.location && window.location.origin;
  if (!origin) {
    return undefined;
  }

  return normalizeBase(`${normalizeBase(origin)}${ensureLeadingSlash(candidate)}`);
};

const resolveApiBaseUrl = () => {
  const explicit = (process.env.REACT_APP_API_URL || '').trim();

  if (typeof window !== 'undefined') {
    const proxyBase = resolveProxyBase();
    if (proxyBase) {
      return proxyBase;
    }

    const location = window.location || {};
    const hostname = location.hostname;
    const origin = location.origin;
    const pathname = location.pathname;
    const hasProxyBootstrap = typeof window.__APP_MONITOR_PROXY_INFO__ !== 'undefined';
    const proxiedPath = isLikelyProxiedPath(pathname);
    const remoteHost = hostname ? !isLocalHostname(hostname) : false;

    if (explicit) {
      const normalized = normalizeBase(explicit);

      if (remoteHost && !hasProxyBootstrap && !proxiedPath && /localhost|127\.0\.0\.1/i.test(normalized)) {
        return normalizeBase(origin || normalized);
      }

      return normalized;
    }

    if (remoteHost && !hasProxyBootstrap && !proxiedPath) {
      return normalizeBase(origin || LOOPBACK_DEFAULT);
    }
  }

  if (explicit) {
    return normalizeBase(explicit);
  }

  return LOOPBACK_DEFAULT;
};

const buildApiUrl = (path) => {
  const base = resolveApiBaseUrl();
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${normalizeBase(base)}${normalizedPath}`;
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

  const engines = [
    { id: 'javascript', name: 'JavaScript', icon: Code, color: 'text-neon-yellow' },
    { id: 'pico8', name: 'PICO-8', icon: Gamepad2, color: 'text-neon-pink' },
    { id: 'tic80', name: 'TIC-80', icon: Terminal, color: 'text-neon-cyan' }
  ];

  const promptTemplates = [
    "Create a simple platformer with a blue character who can jump and collect coins",
    "Make a space shooter where you dodge asteroids and shoot enemies",
    "Build a puzzle game with colored blocks that disappear when matched",
    "Create a retro snake game with neon graphics and power-ups",
    "Make a racing game viewed from above with multiple cars",
    "Build a breakout clone with special power-up bricks",
    "Create a maze game where you collect all dots while avoiding ghosts",
    "Make a tower defense game with robots attacking a base"
  ];

  // API functions
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
      
      // Show success message
      alert(`Game generation started! Generation ID: ${response.data.generation_id}`);
      setGamePrompt('');
      
      // Refresh games list
      fetchGames();
    } catch (err) {
      setError('Failed to start game generation. Please try again.');
      console.error('Error generating game:', err);
    } finally {
      setGeneratingGame(false);
    }
  };

  const playGame = async (gameId) => {
    try {
      await axios.post(buildApiUrl(`/api/games/${gameId}/play`), {});
      // Update play count in local state
      setGames(games.map(game => 
        game.id === gameId 
          ? { ...game, play_count: game.play_count + 1 }
          : game
      ));
    } catch (err) {
      console.error('Error recording play:', err);
    }
  };

  // Effects
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

  // Components
  const LoadingSpinner = () => (
    <div className="flex justify-center items-center py-8">
      <div className="loading-spinner"></div>
      <span className="ml-3 text-neon-cyan font-mono">Loading games...</span>
    </div>
  );

  const ErrorMessage = ({ message }) => (
    <div className="bg-red-900 bg-opacity-30 border border-red-500 border-opacity-50 rounded-lg p-4 mb-4">
      <p className="text-red-400 font-mono">{message}</p>
    </div>
  );

  const GameCard = ({ game, featured = false }) => (
    <div className={`game-card ${featured ? 'border-neon-yellow' : ''} cursor-pointer`}
         onClick={() => setSelectedGame(game)}>
      {featured && (
        <div className="flex items-center mb-2">
          <Star className="w-4 h-4 text-neon-yellow mr-1" />
          <span className="text-neon-yellow text-xs font-mono uppercase">Featured</span>
        </div>
      )}
      
      <h3 className="text-neon-cyan font-bold text-lg mb-2 font-mono">{game.title}</h3>
      
      <p className="text-gray-300 text-sm mb-3 line-clamp-2">
        {game.description || 'No description available'}
      </p>
      
      <div className="flex items-center justify-between mb-3">
        <span className={`px-2 py-1 rounded text-xs font-mono uppercase ${
          engines.find(e => e.id === game.engine)?.color || 'text-neon-cyan'
        }`}>
          {game.engine}
        </span>
        
        <div className="flex items-center space-x-3 text-sm text-gray-400">
          <div className="flex items-center">
            <Eye className="w-4 h-4 mr-1" />
            {game.play_count}
          </div>
          {game.rating && (
            <div className="flex items-center">
              <Star className="w-4 h-4 mr-1 text-neon-yellow" />
              {game.rating.toFixed(1)}
            </div>
          )}
        </div>
      </div>
      
      {game.tags && game.tags.length > 0 && (
        <div className="flex flex-wrap gap-1 mb-3">
          {game.tags.slice(0, 3).map((tag, index) => (
            <span key={index} className="px-2 py-1 bg-neon-purple bg-opacity-20 text-neon-purple text-xs rounded font-mono">
              {tag}
            </span>
          ))}
        </div>
      )}
      
      <button 
        className="neon-button w-full"
        onClick={(e) => {
          e.stopPropagation();
          playGame(game.id);
        }}
      >
        <Play className="w-4 h-4 inline mr-2" />
        Play Game
      </button>
    </div>
  );

  const GameModal = ({ game, onClose }) => {
    if (!game) return null;

    return (
      <div className="fixed inset-0 bg-black bg-opacity-80 flex items-center justify-center z-50 p-4">
        <div className="bg-retro-dark border border-neon-cyan rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto">
          <div className="p-6">
            <div className="flex justify-between items-start mb-4">
              <h2 className="text-2xl font-bold text-neon-cyan font-mono">{game.title}</h2>
              <button 
                onClick={onClose}
                className="text-neon-pink hover:text-white text-2xl"
              >
                ×
              </button>
            </div>
            
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
              <div>
                <h3 className="text-neon-yellow font-mono mb-2">Game Info</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-400">Engine:</span>
                    <span className="text-neon-cyan">{game.engine}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Plays:</span>
                    <span className="text-neon-green">{game.play_count}</span>
                  </div>
                  {game.rating && (
                    <div className="flex justify-between">
                      <span className="text-gray-400">Rating:</span>
                      <span className="text-neon-yellow">{game.rating.toFixed(1)} ★</span>
                    </div>
                  )}
                  <div className="flex justify-between">
                    <span className="text-gray-400">Created:</span>
                    <span className="text-gray-300">{new Date(game.created_at).toLocaleDateString()}</span>
                  </div>
                </div>
              </div>
              
              <div>
                <h3 className="text-neon-yellow font-mono mb-2">Tags</h3>
                <div className="flex flex-wrap gap-1">
                  {game.tags && game.tags.map((tag, index) => (
                    <span key={index} className="px-2 py-1 bg-neon-purple bg-opacity-20 text-neon-purple text-xs rounded font-mono">
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
            </div>
            
            {game.description && (
              <div className="mb-4">
                <h3 className="text-neon-yellow font-mono mb-2">Description</h3>
                <p className="text-gray-300 text-sm">{game.description}</p>
              </div>
            )}
            
            {game.prompt && (
              <div className="mb-6">
                <h3 className="text-neon-yellow font-mono mb-2">Original Prompt</h3>
                <p className="text-gray-300 text-sm bg-retro-purple bg-opacity-20 p-3 rounded font-mono">
                  "{game.prompt}"
                </p>
              </div>
            )}
            
            <div className="flex space-x-3">
              <button 
                className="neon-button flex-1"
                onClick={() => playGame(game.id)}
              >
                <Play className="w-4 h-4 inline mr-2" />
                Play Game
              </button>
              <button className="neon-button border-neon-pink text-neon-pink hover:bg-neon-pink">
                <Zap className="w-4 h-4 inline mr-2" />
                Remix
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-retro-gradient retro-grid">
      {/* Header */}
      <header className="border-b border-neon-cyan border-opacity-30 bg-retro-dark bg-opacity-80 backdrop-blur-sm">
        <div className="container mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-4xl font-bold neon-text font-mono glitch-text" data-text="RETRO GAMES">
                RETRO GAMES
              </h1>
              <p className="text-neon-green font-mono mt-1">AI-Powered Game Launcher</p>
            </div>
            
            <div className="flex items-center space-x-4">
              <div className="text-neon-cyan font-mono text-sm">
                <Cpu className="w-4 h-4 inline mr-1" />
                SYSTEM ONLINE
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-retro-dark bg-opacity-60 backdrop-blur-sm border-b border-neon-cyan border-opacity-20">
        <div className="container mx-auto px-4">
          <div className="flex space-x-8">
            {[
              { id: 'browse', label: 'Browse Games', icon: Gamepad2 },
              { id: 'generate', label: 'Generate Game', icon: Brain },
              { id: 'featured', label: 'Featured', icon: Star },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center py-4 px-2 font-mono border-b-2 transition-colors ${
                  activeTab === tab.id
                    ? 'border-neon-cyan text-neon-cyan'
                    : 'border-transparent text-gray-400 hover:text-neon-cyan'
                }`}
              >
                <tab.icon className="w-4 h-4 mr-2" />
                {tab.label}
              </button>
            ))}
          </div>
        </div>
      </nav>

      {/* Content */}
      <main className="container mx-auto px-4 py-8">
        {error && <ErrorMessage message={error} />}

        {/* Browse Tab */}
        {activeTab === 'browse' && (
          <>
            {/* Search */}
            <div className="mb-8">
              <div className="relative max-w-md mx-auto">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-neon-green w-5 h-5" />
                <input
                  type="text"
                  placeholder="Search retro games..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="search-input w-full pl-10"
                />
              </div>
            </div>

            {/* Games Grid */}
            {loading ? (
              <LoadingSpinner />
            ) : games.length > 0 ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                {games.map((game) => (
                  <GameCard key={game.id} game={game} />
                ))}
              </div>
            ) : (
              <div className="text-center py-12">
                <Gamepad2 className="w-16 h-16 text-neon-cyan mx-auto mb-4 opacity-50" />
                <h3 className="text-xl text-neon-cyan font-mono mb-2">No Games Found</h3>
                <p className="text-gray-400">Try generating some new games or adjusting your search.</p>
              </div>
            )}
          </>
        )}

        {/* Generate Tab */}
        {activeTab === 'generate' && (
          <div className="max-w-2xl mx-auto">
            <div className="bg-retro-purple bg-opacity-20 border border-neon-pink border-opacity-50 rounded-lg p-6">
              <h2 className="text-2xl font-bold text-neon-pink font-mono mb-6 flex items-center">
                <Brain className="w-6 h-6 mr-2" />
                AI Game Generator
              </h2>
              
              {/* Engine Selection */}
              <div className="mb-6">
                <label className="block text-neon-yellow font-mono mb-3">Game Engine:</label>
                <div className="grid grid-cols-3 gap-3">
                  {engines.map((engine) => (
                    <button
                      key={engine.id}
                      onClick={() => setSelectedEngine(engine.id)}
                      className={`flex items-center justify-center py-3 px-4 rounded font-mono border-2 transition-all ${
                        selectedEngine === engine.id
                          ? 'border-neon-cyan bg-neon-cyan bg-opacity-20 text-neon-cyan'
                          : 'border-gray-600 text-gray-400 hover:border-neon-cyan hover:text-neon-cyan'
                      }`}
                    >
                      <engine.icon className="w-4 h-4 mr-2" />
                      {engine.name}
                    </button>
                  ))}
                </div>
              </div>

              {/* Prompt Input */}
              <div className="mb-6">
                <label className="block text-neon-yellow font-mono mb-3">Game Description:</label>
                <textarea
                  value={gamePrompt}
                  onChange={(e) => setGamePrompt(e.target.value)}
                  placeholder="Describe the game you want to create..."
                  className="w-full h-32 bg-retro-dark bg-opacity-80 border border-neon-green border-opacity-50 text-neon-green placeholder-neon-green placeholder-opacity-50 px-4 py-3 rounded font-mono resize-none focus:outline-none focus:border-neon-green focus:border-opacity-100"
                />
              </div>

              {/* Template Suggestions */}
              <div className="mb-6">
                <label className="block text-neon-yellow font-mono mb-3">Try These Ideas:</label>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                  {promptTemplates.slice(0, 4).map((template, index) => (
                    <button
                      key={index}
                      onClick={() => setGamePrompt(template)}
                      className="text-left p-3 bg-retro-dark bg-opacity-40 border border-gray-600 rounded text-sm text-gray-300 hover:border-neon-cyan hover:text-neon-cyan transition-colors font-mono"
                    >
                      {template}
                    </button>
                  ))}
                </div>
              </div>

              {/* Generate Button */}
              <button
                onClick={generateGame}
                disabled={!gamePrompt.trim() || generatingGame}
                className="neon-button w-full py-4 text-lg disabled:opacity-50 disabled:cursor-not-allowed"
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
          </div>
        )}

        {/* Featured Tab */}
        {activeTab === 'featured' && (
          <>
            <div className="text-center mb-8">
              <h2 className="text-3xl font-bold text-neon-yellow font-mono mb-2">Featured Games</h2>
              <p className="text-gray-400">Top-rated games from our community</p>
            </div>

            {featuredGames.length > 0 ? (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {featuredGames.map((game) => (
                  <GameCard key={game.id} game={game} featured={true} />
                ))}
              </div>
            ) : (
              <div className="text-center py-12">
                <Trophy className="w-16 h-16 text-neon-yellow mx-auto mb-4 opacity-50" />
                <h3 className="text-xl text-neon-yellow font-mono mb-2">No Featured Games Yet</h3>
                <p className="text-gray-400">Start creating and rating games to see featured content!</p>
              </div>
            )}
          </>
        )}
      </main>

      {/* Game Modal */}
      {selectedGame && (
        <GameModal 
          game={selectedGame} 
          onClose={() => setSelectedGame(null)} 
        />
      )}

      {/* Footer */}
      <footer className="border-t border-neon-cyan border-opacity-30 bg-retro-dark bg-opacity-80 backdrop-blur-sm mt-16">
        <div className="container mx-auto px-4 py-6">
          <div className="text-center">
            <p className="text-neon-cyan font-mono">
              <Rocket className="w-4 h-4 inline mr-1" />
              Powered by AI • Built with React & Tailwind
            </p>
            <p className="text-gray-400 text-sm mt-1">
              Create, play, and share retro games with artificial intelligence
            </p>
          </div>
        </div>
      </footer>

      {/* Ambient effects */}
      <div className="fixed inset-0 pointer-events-none">
        <div className="absolute top-0 left-0 w-full h-px bg-gradient-to-r from-transparent via-neon-cyan to-transparent opacity-50 animate-pulse"></div>
        <div className="absolute bottom-0 left-0 w-full h-px bg-gradient-to-r from-transparent via-neon-pink to-transparent opacity-50 animate-pulse"></div>
      </div>
    </div>
  );
}

export default App;