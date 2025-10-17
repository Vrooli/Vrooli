import React from 'react';
import { motion } from 'framer-motion';
import clsx from 'clsx';

const ScenarioTile = ({ scenario, onClick }) => {
  const {
    title = 'Mystery Game',
    description = 'Something fun awaits!',
    icon = '❓',
    color = 'bg-gradient-to-br from-purple-400 to-blue-400',
    category = 'fun',
    disabled = false
  } = scenario;

  // Category-based decorations
  const getCategoryDecoration = () => {
    switch (category) {
      case 'games':
        return '🎮';
      case 'learn':
        return '📚';
      case 'create':
        return '🎨';
      case 'music':
        return '🎵';
      case 'stories':
        return '📖';
      default:
        return '⭐';
    }
  };

  return (
    <motion.div
      whileHover={!disabled ? { scale: 1.05, rotate: [-1, 1, -1, 0] } : {}}
      whileTap={!disabled ? { scale: 0.95 } : {}}
      className={clsx(
        'scenario-tile',
        color,
        disabled && 'opacity-50 cursor-not-allowed'
      )}
      onClick={onClick}
    >
      {/* Category badge */}
      <div className="absolute top-2 right-2 bg-white/30 backdrop-blur-sm rounded-full p-2 text-2xl">
        {getCategoryDecoration()}
      </div>

      {/* Main icon */}
      <motion.div
        className="text-7xl mb-4"
        animate={{ 
          rotate: disabled ? 0 : [0, -10, 10, -10, 10, 0],
        }}
        transition={{ 
          duration: 2,
          repeat: Infinity,
          repeatDelay: 3
        }}
      >
        {icon}
      </motion.div>

      {/* Title */}
      <h3 className="text-2xl font-fredoka font-bold text-white text-shadow mb-2 text-center">
        {title}
      </h3>

      {/* Description */}
      <p className="text-white/90 text-center text-sm">
        {description}
      </p>

      {/* Fun hover effect particles */}
      {!disabled && (
        <>
          <motion.div
            className="absolute top-2 left-2 text-2xl opacity-0"
            whileHover={{ opacity: 1 }}
            animate={{ rotate: 360 }}
            transition={{ duration: 2, repeat: Infinity }}
          >
            ✨
          </motion.div>
          <motion.div
            className="absolute bottom-2 right-2 text-2xl opacity-0"
            whileHover={{ opacity: 1 }}
            animate={{ rotate: -360 }}
            transition={{ duration: 2, repeat: Infinity }}
          >
            ✨
          </motion.div>
        </>
      )}
    </motion.div>
  );
};

export default ScenarioTile;