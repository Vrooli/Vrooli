import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';

const Mascot = ({ message }) => {
  const [isVisible, setIsVisible] = useState(true);
  const [currentMessage, setCurrentMessage] = useState(message);
  const [isDismissed, setIsDismissed] = useState(false);

  useEffect(() => {
    if (isDismissed || message === currentMessage) {
      return;
    }

    setIsVisible(false);

    const timeoutId = setTimeout(() => {
      setCurrentMessage(message);
      setIsVisible(true);
    }, 300);

    return () => clearTimeout(timeoutId);
  }, [message, currentMessage, isDismissed]);

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
    <div className="fixed bottom-4 right-4 sm:bottom-8 sm:right-8 z-50">
      <div className="pointer-events-none">
        <AnimatePresence>
          {isDismissed && (
            <motion.button
              key="restore"
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.8 }}
              transition={{ type: "spring", stiffness: 260, damping: 20 }}
              onClick={() => setIsDismissed(false)}
              className="pointer-events-auto flex items-center gap-2 rounded-full bg-white px-4 py-2 shadow-lg border border-kid-blue/40 text-sm font-bold text-kid-blue"
              aria-label="Show Robo Buddy again"
            >
              <span role="img" aria-hidden="true">🤖</span>
              Bring Robo Back
            </motion.button>
          )}
        </AnimatePresence>

        <AnimatePresence mode="wait">
          {!isDismissed && isVisible && (
            <motion.div
              key="mascot"
              initial={{ scale: 0, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0, opacity: 0 }}
              transition={{ type: "spring", bounce: 0.5 }}
              className="relative pointer-events-auto"
            >
              {/* Dismiss control lives on the speech bubble so kids can tuck Robo away */}
              <motion.div
                initial={{ y: 20, opacity: 0 }}
                animate={{ y: 0, opacity: 1 }}
                transition={{ delay: 0.2 }}
                className="absolute bottom-24 sm:bottom-32 right-0 bg-white rounded-3xl px-4 py-5 shadow-xl min-w-[180px] max-w-[260px] sm:min-w-[200px] sm:max-w-[300px]"
              >
                <button
                  type="button"
                  onClick={() => setIsDismissed(true)}
                  className="absolute -top-3 -right-3 w-8 h-8 rounded-full bg-kid-red text-white flex items-center justify-center shadow-md"
                  aria-label="Hide Robo Buddy"
                >
                  ×
                </button>
                <p className="text-gray-800 font-comic font-bold text-base sm:text-lg">
                  {currentMessage}
                </p>
                <div className="absolute -bottom-2 right-6 sm:right-8 w-4 h-4 bg-white transform rotate-45"></div>
              </motion.div>

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
                <div className="w-24 h-24 sm:w-32 sm:h-32 relative">
                  <div className="absolute inset-0 bg-gradient-to-br from-kid-purple via-kid-blue to-kid-teal rounded-3xl shadow-2xl">
                    <div className="absolute top-3 sm:top-4 left-3 right-3 sm:left-4 sm:right-4 h-12 sm:h-16 bg-gray-900 rounded-2xl flex items-center justify-center">
                      <motion.div
                        className="flex gap-2 sm:gap-3"
                        animate={{ scale: [1, 1.2, 1] }}
                        transition={{ duration: 3, repeat: Infinity }}
                      >
                        <div className="w-3 h-3 sm:w-4 sm:h-4 bg-kid-green rounded-full animate-pulse"></div>
                        <div className="w-3 h-3 sm:w-4 sm:h-4 bg-kid-green rounded-full animate-pulse"></div>
                      </motion.div>
                    </div>

                    <div className="absolute bottom-3 sm:bottom-4 left-3 right-3 sm:left-4 sm:right-4 h-3 sm:h-4 bg-gray-800 rounded-full flex items-center justify-center">
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

                  <motion.div
                    className="absolute -top-2 left-1/2 transform -translate-x-1/2"
                    animate={{ rotate: [0, 360] }}
                    transition={{ duration: 10, repeat: Infinity, ease: "linear" }}
                  >
                    <div className="w-1.5 h-5 sm:w-2 sm:h-6 bg-gray-600 rounded-full"></div>
                    <div className="absolute -top-2 left-1/2 transform -translate-x-1/2 w-3 h-3 sm:w-4 sm:h-4 bg-kid-red rounded-full animate-pulse"></div>
                  </motion.div>

                  <motion.div
                    className="absolute top-6 sm:top-8 -left-2 sm:-left-3 w-5 sm:w-6 h-12 sm:h-16 bg-gradient-to-b from-kid-purple to-kid-blue rounded-full"
                    animate={{ rotate: [-10, 10, -10] }}
                    transition={{ duration: 2, repeat: Infinity }}
                  ></motion.div>
                  <motion.div
                    className="absolute top-6 sm:top-8 -right-2 sm:-right-3 w-5 sm:w-6 h-12 sm:h-16 bg-gradient-to-b from-kid-purple to-kid-blue rounded-full"
                    animate={{ rotate: [10, -10, 10] }}
                    transition={{ duration: 2, repeat: Infinity }}
                  ></motion.div>
                </div>
              </motion.div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
};

export default Mascot;
