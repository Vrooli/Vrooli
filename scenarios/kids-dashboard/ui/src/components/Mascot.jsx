import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';

const Mascot = ({ message }) => {
  const [isVisible, setIsVisible] = useState(true);
  const [currentMessage, setCurrentMessage] = useState(message);

  useEffect(() => {
    if (message !== currentMessage) {
      setIsVisible(false);
      setTimeout(() => {
        setCurrentMessage(message);
        setIsVisible(true);
      }, 300);
    }
  }, [message, currentMessage]);

  // Random encouragements when idle
  const randomMessages = [
    "You're doing great! 🌟",
    "What an explorer you are! 🚀",
    "Having fun? I am! 🎉",
    "You're awesome! 💫",
    "Great choices today! 👍",
    "Keep being amazing! 🌈"
  ];

  return (
    <div className="fixed bottom-8 right-8 z-50 pointer-events-none">
      <AnimatePresence mode="wait">
        {isVisible && (
          <motion.div
            initial={{ scale: 0, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0, opacity: 0 }}
            transition={{ type: "spring", bounce: 0.5 }}
            className="relative"
          >
            {/* Speech bubble */}
            <motion.div
              initial={{ y: 20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              transition={{ delay: 0.2 }}
              className="absolute bottom-32 right-0 bg-white rounded-3xl p-4 shadow-xl min-w-[200px] max-w-[300px]"
            >
              <p className="text-gray-800 font-comic font-bold text-lg">
                {currentMessage}
              </p>
              {/* Speech bubble tail */}
              <div className="absolute -bottom-2 right-8 w-4 h-4 bg-white transform rotate-45"></div>
            </motion.div>

            {/* Mascot character - Vrooli the Robot */}
            <motion.div
              animate={{ 
                y: [0, -10, 0],
                rotate: [0, 5, -5, 0]
              }}
              transition={{ 
                duration: 4,
                repeat: Infinity,
                repeatType: "reverse"
              }}
              className="relative"
            >
              {/* Robot body */}
              <div className="w-32 h-32 relative">
                {/* Main body */}
                <div className="absolute inset-0 bg-gradient-to-br from-kid-purple via-kid-blue to-kid-teal rounded-3xl shadow-2xl">
                  {/* Face screen */}
                  <div className="absolute top-4 left-4 right-4 h-16 bg-gray-900 rounded-2xl flex items-center justify-center">
                    {/* Eyes */}
                    <motion.div
                      className="flex gap-3"
                      animate={{ scale: [1, 1.2, 1] }}
                      transition={{ duration: 3, repeat: Infinity }}
                    >
                      <div className="w-4 h-4 bg-kid-green rounded-full animate-pulse"></div>
                      <div className="w-4 h-4 bg-kid-green rounded-full animate-pulse"></div>
                    </motion.div>
                  </div>
                  
                  {/* Mouth/speaker */}
                  <div className="absolute bottom-4 left-4 right-4 h-4 bg-gray-800 rounded-full flex items-center justify-center">
                    <motion.div
                      className="flex gap-1"
                      animate={{ opacity: [0.5, 1, 0.5] }}
                      transition={{ duration: 1, repeat: Infinity }}
                    >
                      <div className="w-1 h-2 bg-kid-yellow rounded-full"></div>
                      <div className="w-1 h-3 bg-kid-yellow rounded-full"></div>
                      <div className="w-1 h-2 bg-kid-yellow rounded-full"></div>
                    </motion.div>
                  </div>
                </div>

                {/* Antenna */}
                <motion.div
                  className="absolute -top-2 left-1/2 transform -translate-x-1/2"
                  animate={{ rotate: [0, 360] }}
                  transition={{ duration: 10, repeat: Infinity, ease: "linear" }}
                >
                  <div className="w-2 h-6 bg-gray-600 rounded-full"></div>
                  <div className="absolute -top-2 left-1/2 transform -translate-x-1/2 w-4 h-4 bg-kid-red rounded-full animate-pulse"></div>
                </motion.div>

                {/* Arms */}
                <motion.div
                  className="absolute top-8 -left-3 w-6 h-16 bg-gradient-to-b from-kid-purple to-kid-blue rounded-full"
                  animate={{ rotate: [-10, 10, -10] }}
                  transition={{ duration: 2, repeat: Infinity }}
                ></motion.div>
                <motion.div
                  className="absolute top-8 -right-3 w-6 h-16 bg-gradient-to-b from-kid-purple to-kid-blue rounded-full"
                  animate={{ rotate: [10, -10, 10] }}
                  transition={{ duration: 2, repeat: Infinity }}
                ></motion.div>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default Mascot;