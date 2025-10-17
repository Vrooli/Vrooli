// Utility functions for fetching and filtering kid-friendly scenarios

/**
 * Fetches and filters scenarios that are appropriate for kids
 * @returns {Promise<Array>} Array of kid-friendly scenarios
 */
export async function fetchKidFriendlyScenarios() {
  try {
    // In production, this would fetch from the API endpoint
    // For now, we'll simulate with some example data and check local scenarios
    
    // Try to fetch from the local catalog
    const response = await fetch('/api/v1/kids/scenarios').catch(() => null);
    
    if (response && response.ok) {
      const data = await response.json();
      return data.scenarios;
    }
    
    // Fallback to mock data for development/testing
    return getMockScenarios();
  } catch (error) {
    console.error('Error fetching scenarios:', error);
    return getMockScenarios();
  }
}

/**
 * Mock scenarios for testing/development
 */
function getMockScenarios() {
  return [
    {
      id: 'retro-game-launcher',
      name: 'retro-game-launcher',
      title: 'Retro Games',
      description: 'Play classic arcade games!',
      icon: '🕹️',
      color: 'bg-gradient-to-br from-purple-500 to-pink-500',
      category: 'games',
      port: 3301,
      ageRange: '5-12'
    },
    {
      id: 'picker-wheel',
      name: 'picker-wheel',
      title: 'Picker Wheel',
      description: 'Spin the wheel for fun choices!',
      icon: '🎯',
      color: 'bg-gradient-to-br from-yellow-400 to-orange-500',
      category: 'games',
      port: 3302,
      ageRange: '5-12'
    },
    {
      id: 'word-games',
      name: 'word-games',
      title: 'Word Games',
      description: 'Fun puzzles with letters and words!',
      icon: '📝',
      color: 'bg-gradient-to-br from-green-400 to-blue-500',
      category: 'learn',
      port: 3303,
      ageRange: '9-12'
    },
    {
      id: 'study-buddy',
      name: 'study-buddy',
      title: 'Study Buddy',
      description: 'A friendly helper for homework!',
      icon: '📚',
      color: 'bg-gradient-to-br from-teal-400 to-cyan-500',
      category: 'learn',
      port: 3304,
      ageRange: '9-12'
    }
  ];
}

/**
 * Filters scenarios by age range
 * @param {Array} scenarios - Array of scenarios
 * @param {string} ageRange - Age range to filter by (e.g., '5-8', '9-12', '13+')
 * @returns {Array} Filtered scenarios
 */
export function filterByAgeRange(scenarios, ageRange) {
  if (!ageRange) return scenarios;
  
  return scenarios.filter(scenario => {
    if (!scenario.ageRange) return true; // Include if no age range specified
    
    // Simple age range matching
    // In production, this would be more sophisticated
    return scenario.ageRange === ageRange || scenario.ageRange === '5-12';
  });
}

/**
 * Filters scenarios by category
 * @param {Array} scenarios - Array of scenarios
 * @param {string} category - Category to filter by (e.g., 'games', 'learn', 'create')
 * @returns {Array} Filtered scenarios
 */
export function filterByCategory(scenarios, category) {
  if (!category) return scenarios;
  
  return scenarios.filter(scenario => scenario.category === category);
}

/**
 * Checks if a scenario is safe for kids based on its metadata
 * @param {Object} scenario - Scenario object
 * @returns {boolean} True if scenario is kid-friendly
 */
export function isKidFriendly(scenario) {
  // Check for explicit kid-friendly tag
  if (scenario.category && scenario.category.includes('kid-friendly')) {
    return true;
  }
  
  // Check for blacklisted categories
  const blacklist = ['system', 'development', 'admin', 'financial', 'debug'];
  if (scenario.category && blacklist.some(cat => scenario.category.includes(cat))) {
    return false;
  }
  
  // Check metadata tags
  if (scenario.metadata?.tags) {
    const tags = scenario.metadata.tags;
    if (tags.includes('kid-friendly') || tags.includes('kids') || tags.includes('children')) {
      return true;
    }
    if (blacklist.some(cat => tags.includes(cat))) {
      return false;
    }
  }
  
  // Default to false for safety
  return false;
}

/**
 * Sorts scenarios by popularity or alphabetically
 * @param {Array} scenarios - Array of scenarios
 * @param {string} sortBy - Sort method ('popular', 'alphabetical')
 * @returns {Array} Sorted scenarios
 */
export function sortScenarios(scenarios, sortBy = 'alphabetical') {
  const sorted = [...scenarios];
  
  switch (sortBy) {
    case 'popular':
      return sorted.sort((a, b) => (b.popularity || 0) - (a.popularity || 0));
    case 'alphabetical':
    default:
      return sorted.sort((a, b) => a.title.localeCompare(b.title));
  }
}