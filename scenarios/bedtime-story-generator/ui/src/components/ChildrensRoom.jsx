import React from 'react';
import { motion } from 'framer-motion';

const ChildrensRoom = ({ timeOfDay, theme, children }) => {
  const getSunMoonPosition = () => {
    const hour = new Date().getHours();
    if (hour >= 6 && hour < 18) {
      // Sun during day
      const progress = (hour - 6) / 12;
      return {
        type: 'sun',
        x: 20 + progress * 60,
        y: 20 + Math.sin(progress * Math.PI) * -30
      };
    } else {
      // Moon at night
      return {
        type: 'moon',
        x: 50,
        y: 20
      };
    }
  };

  const celestialBody = getSunMoonPosition();

  return (
    <div className={`childrens-room ${theme.class}`}>
      <div className="room-background" style={{ background: theme.background }}>
        {/* Window */}
        <div className="window">
          <div className="window-view">
            {celestialBody.type === 'sun' ? (
              <motion.div
                className="sun"
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                style={{
                  position: 'absolute',
                  left: `${celestialBody.x}%`,
                  top: `${celestialBody.y}%`,
                  width: '40px',
                  height: '40px',
                  background: 'radial-gradient(circle, #ffeb3b 0%, #ffc107 100%)',
                  borderRadius: '50%',
                  boxShadow: '0 0 40px rgba(255, 235, 59, 0.5)'
                }}
              />
            ) : (
              <motion.div
                className="moon"
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                style={{
                  position: 'absolute',
                  left: `${celestialBody.x}%`,
                  top: `${celestialBody.y}%`,
                  width: '35px',
                  height: '35px',
                  background: 'radial-gradient(circle, #f5f5f5 0%, #e0e0e0 100%)',
                  borderRadius: '50%',
                  boxShadow: '0 0 30px rgba(255, 255, 255, 0.3)'
                }}
              />
            )}
            
            {/* Stars for nighttime */}
            {timeOfDay === 'night' && (
              <>
                {[...Array(10)].map((_, i) => (
                  <motion.div
                    key={i}
                    className="star"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: [0.3, 1, 0.3] }}
                    transition={{
                      duration: 2 + Math.random() * 2,
                      repeat: Infinity,
                      delay: Math.random() * 2
                    }}
                    style={{
                      position: 'absolute',
                      left: `${10 + Math.random() * 80}%`,
                      top: `${10 + Math.random() * 60}%`,
                      width: '2px',
                      height: '2px',
                      background: 'white',
                      borderRadius: '50%'
                    }}
                  />
                ))}
              </>
            )}
          </div>
          <div className="window-panes">
            <div className="window-pane-h" />
            <div className="window-pane-v" />
          </div>
        </div>

        {/* Lamp */}
        <div className="lamp">
          <div className="lamp-base" />
          <div className="lamp-shade" />
          <div className="lamp-light" />
        </div>

        {/* Nightlight */}
        <div className="nightlight" />

        {/* Toys and decorations */}
        <motion.div
          className="teddy-bear"
          initial={{ rotate: -5 }}
          animate={{ rotate: 5 }}
          transition={{ duration: 3, repeat: Infinity, repeatType: 'reverse' }}
          style={{
            position: 'absolute',
            bottom: '15%',
            right: '10%',
            fontSize: '3rem'
          }}
        >
          🧸
        </motion.div>

        <div
          className="toy-blocks"
          style={{
            position: 'absolute',
            bottom: '10%',
            left: '10%',
            fontSize: '2rem'
          }}
        >
          🧱
        </div>

        <motion.div
          className="ball"
          animate={{ y: [0, -10, 0] }}
          transition={{ duration: 2, repeat: Infinity }}
          style={{
            position: 'absolute',
            bottom: '12%',
            left: '30%',
            fontSize: '2rem'
          }}
        >
          ⚽
        </motion.div>
      </div>

      {/* Main content area */}
      <div className="room-content">
        {children}
      </div>
    </div>
  );
};

export default ChildrensRoom;