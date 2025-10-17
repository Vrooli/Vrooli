import React from 'react';
import ScenarioTile from './ScenarioTile';
import { motion } from 'framer-motion';
import { Sparkles } from 'lucide-react';

const Dashboard = ({ scenarios, onSelectScenario }) => {
  // Default scenarios to show when none are available
  const placeholderScenarios = [
    {
      id: 'coming-soon-1',
      title: 'More Fun Coming Soon!',
      description: 'Ask a grown-up to add games!',
      icon: '🎮',
      color: 'bg-gradient-to-br from-purple-400 to-pink-400',
      disabled: true
    }
  ];

  const displayScenarios = scenarios.length > 0 ? scenarios : placeholderScenarios;

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="min-h-screen p-8"
    >
      {/* Header */}
      <motion.header
        initial={{ y: -50 }}
        animate={{ y: 0 }}
        className="text-center mb-12"
      >
        <h1 className="text-6xl md:text-8xl font-fredoka font-bold text-transparent bg-clip-text bg-gradient-to-r from-kid-red via-kid-yellow to-kid-teal text-shadow mb-4">
          Vrooli Kids Zone
        </h1>
        <div className="flex justify-center items-center gap-4 text-2xl">
          <Sparkles className="text-kid-yellow animate-pulse" size={32} />
          <span className="text-gray-700 font-semibold">Choose Your Adventure!</span>
          <Sparkles className="text-kid-yellow animate-pulse" size={32} />
        </div>
      </motion.header>

      {/* Scenario Grid */}
      <div className="max-w-7xl mx-auto">
        <motion.div 
          className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
          initial={{ scale: 0.8 }}
          animate={{ scale: 1 }}
          transition={{ delay: 0.2 }}
        >
          {displayScenarios.map((scenario, index) => (
            <motion.div
              key={scenario.id}
              initial={{ opacity: 0, y: 50 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.1 }}
            >
              <ScenarioTile
                scenario={scenario}
                onClick={() => !scenario.disabled && onSelectScenario(scenario)}
              />
            </motion.div>
          ))}
        </motion.div>
      </div>

      {/* Fun footer decorations */}
      <div className="fixed bottom-0 left-0 right-0 h-32 pointer-events-none">
        <div className="relative h-full">
          <motion.div
            className="absolute bottom-0 left-10 text-4xl"
            animate={{ y: [0, -10, 0] }}
            transition={{ duration: 2, repeat: Infinity }}
          >
            🦄
          </motion.div>
          <motion.div
            className="absolute bottom-0 right-10 text-4xl"
            animate={{ y: [0, -15, 0] }}
            transition={{ duration: 2.5, repeat: Infinity, delay: 0.5 }}
          >
            🚀
          </motion.div>
          <motion.div
            className="absolute bottom-0 left-1/2 transform -translate-x-1/2 text-4xl"
            animate={{ y: [0, -20, 0] }}
            transition={{ duration: 3, repeat: Infinity, delay: 1 }}
          >
            🎨
          </motion.div>
        </div>
      </div>
    </motion.div>
  );
};

export default Dashboard;