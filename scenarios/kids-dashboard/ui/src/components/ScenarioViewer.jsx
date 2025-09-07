import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { ArrowLeft, Home, RefreshCw, Maximize2, Minimize2 } from 'lucide-react';

const ScenarioViewer = ({ scenario, onBack }) => {
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [iframeKey, setIframeKey] = useState(0);

  // Build the scenario URL
  const getScenarioUrl = () => {
    // This will be the URL where the scenario is hosted
    // For now, we'll use a relative path that assumes scenarios run on different ports
    const basePort = parseInt(scenario.port || '3000');
    return `http://localhost:${basePort}`;
  };

  const handleRefresh = () => {
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
          
          <iframe
            key={iframeKey}
            src={getScenarioUrl()}
            className="w-full h-full border-0"
            title={scenario.title}
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
            onLoad={() => setIsLoading(false)}
            style={{ minHeight: 'calc(100vh - 200px)' }}
          />
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