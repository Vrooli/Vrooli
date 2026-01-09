import React, { useState, useEffect } from 'react';
import Dashboard from './components/Dashboard';
import ScenarioViewer from './components/ScenarioViewer';
import Mascot from './components/Mascot';
import { fetchKidFriendlyScenarios } from './utils/scenarios';
import { motion, AnimatePresence } from 'framer-motion';

function App() {
  const [scenarios, setScenarios] = useState([]);
  const [currentScenario, setCurrentScenario] = useState(null);
  const [loading, setLoading] = useState(true);
  const [mascotMessage, setMascotMessage] = useState("Hi there! Pick something fun to play with!");

  useEffect(() => {
    loadScenarios();
  }, []);

  const loadScenarios = async () => {
    try {
      const kidScenarios = await fetchKidFriendlyScenarios();
      setScenarios(kidScenarios);
      setLoading(false);
      
      if (kidScenarios.length === 0) {
        setMascotMessage("Hmm, no games ready yet! Ask a grown-up to add some!");
      }
    } catch (error) {
      console.error('Error loading scenarios:', error);
      setLoading(false);
      setMascotMessage("Oops! Something went wrong. Let's try again!");
    }
  };

  const handleScenarioSelect = (scenario) => {
    setCurrentScenario(scenario);
    setMascotMessage(`Great choice! Have fun with ${scenario.title}!`);
  };

  const handleBackToDashboard = () => {
    setCurrentScenario(null);
    setMascotMessage("Welcome back! What do you want to play next?");
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <motion.div
          animate={{ rotate: 360 }}
          transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
          className="w-32 h-32"
        >
          <div className="rainbow-bg w-full h-full rounded-full flex items-center justify-center text-white text-6xl">
            🤖
          </div>
        </motion.div>
      </div>
    );
  }

  return (
    <div className="min-h-screen relative overflow-hidden">
      {/* Background decorations */}
      <div className="absolute inset-0 bg-pattern-dots opacity-10 pointer-events-none" aria-hidden="true"></div>
      
      {/* Floating background elements */}
      <motion.div
        className="absolute top-10 left-10 text-6xl pointer-events-none"
        animate={{ 
          y: [0, -20, 0],
          rotate: [0, 10, -10, 0]
        }}
        transition={{ duration: 5, repeat: Infinity }}
      >
        🌟
      </motion.div>
      
      <motion.div
        className="absolute top-20 right-20 text-5xl pointer-events-none"
        animate={{ 
          y: [0, 20, 0],
          rotate: [0, -15, 15, 0]
        }}
        transition={{ duration: 4, repeat: Infinity, delay: 1 }}
      >
        🎈
      </motion.div>
      
      <motion.div
        className="absolute bottom-20 left-20 text-5xl pointer-events-none"
        animate={{ 
          x: [0, 20, 0],
          y: [0, -10, 0]
        }}
        transition={{ duration: 6, repeat: Infinity, delay: 2 }}
      >
        🌈
      </motion.div>

      {/* Main content */}
      <AnimatePresence mode="wait">
        {currentScenario ? (
          <ScenarioViewer
            key="viewer"
            scenario={currentScenario}
            onBack={handleBackToDashboard}
          />
        ) : (
          <Dashboard
            key="dashboard"
            scenarios={scenarios}
            onSelectScenario={handleScenarioSelect}
          />
        )}
      </AnimatePresence>

      {/* Mascot */}
      <Mascot message={mascotMessage} />
    </div>
  );
}

export default App;
