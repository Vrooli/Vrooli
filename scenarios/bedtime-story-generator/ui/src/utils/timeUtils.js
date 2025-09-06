export const getTimeOfDay = () => {
  const hour = new Date().getHours();
  
  if (hour >= 6 && hour < 18) {
    return 'day';
  } else if (hour >= 18 && hour < 21) {
    return 'evening';
  } else {
    return 'night';
  }
};

export const getThemeForTime = () => {
  const timeOfDay = getTimeOfDay();
  
  const themes = {
    day: {
      class: 'theme-day',
      background: 'linear-gradient(180deg, #87CEEB 0%, #98D8E8 100%)',
      textColor: '#2c3e50',
      shadowColor: 'rgba(0, 0, 0, 0.1)',
      windowGlow: 'rgba(255, 255, 200, 0.5)',
      lampOpacity: 0,
      nightlightOpacity: 0,
      sunMoonType: 'sun',
      celestialColor: '#ffeb3b'
    },
    evening: {
      class: 'theme-evening',
      background: 'linear-gradient(180deg, #ff9a56 0%, #ff6b6b 100%)',
      textColor: '#3d2314',
      shadowColor: 'rgba(0, 0, 0, 0.3)',
      windowGlow: 'rgba(255, 180, 100, 0.6)',
      lampOpacity: 0.8,
      nightlightOpacity: 0,
      sunMoonType: 'sun-setting',
      celestialColor: '#ff9a56'
    },
    night: {
      class: 'theme-night',
      background: 'linear-gradient(180deg, #1a1a2e 0%, #0f0f1e 100%)',
      textColor: '#e0e0e0',
      shadowColor: 'rgba(0, 0, 0, 0.5)',
      windowGlow: 'rgba(100, 100, 200, 0.3)',
      lampOpacity: 0.3,
      nightlightOpacity: 1,
      sunMoonType: 'moon',
      celestialColor: '#f5f5f5'
    }
  };
  
  return themes[timeOfDay];
};

export const getGreeting = () => {
  const hour = new Date().getHours();
  
  if (hour >= 5 && hour < 12) {
    return 'Good morning! ☀️';
  } else if (hour >= 12 && hour < 17) {
    return 'Good afternoon! 🌤️';
  } else if (hour >= 17 && hour < 21) {
    return 'Good evening! 🌅';
  } else {
    return 'Time for bed! 🌙';
  }
};

export const isBedroomTime = () => {
  const hour = new Date().getHours();
  return hour >= 19 || hour < 7;
};

export const getReadingTimeMessage = (minutes) => {
  if (minutes <= 5) {
    return 'A quick bedtime tale';
  } else if (minutes <= 10) {
    return 'Perfect for one bedtime story';
  } else {
    return 'A longer adventure for tonight';
  }
};