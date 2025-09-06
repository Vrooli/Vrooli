import React, { useState } from 'react';
import { FiChevronDown, FiCheck } from 'react-icons/fi';
import clsx from 'clsx';

const ScenarioSelector = ({ scenarios, selectedScenario, onSelect }) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center space-x-2 px-4 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-primary-500"
      >
        <span className="text-gray-900 dark:text-white">
          {selectedScenario ? selectedScenario.display_name : 'Select Scenario'}
        </span>
        <FiChevronDown className={clsx(
          'transition-transform',
          isOpen && 'rotate-180'
        )} />
      </button>

      {isOpen && (
        <div className="absolute top-full left-0 mt-1 w-64 bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg shadow-lg z-50">
          <div className="py-1">
            {scenarios.map(scenario => (
              <button
                key={scenario.id}
                onClick={() => {
                  onSelect(scenario);
                  setIsOpen(false);
                }}
                className={clsx(
                  'w-full text-left px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-600 flex items-center justify-between',
                  selectedScenario?.id === scenario.id && 'bg-gray-50 dark:bg-gray-600'
                )}
              >
                <div>
                  <div className="font-medium text-gray-900 dark:text-white">
                    {scenario.display_name}
                  </div>
                  {scenario.description && (
                    <div className="text-sm text-gray-500 dark:text-gray-400 line-clamp-1">
                      {scenario.description}
                    </div>
                  )}
                </div>
                {selectedScenario?.id === scenario.id && (
                  <FiCheck className="text-primary-600 dark:text-primary-400" />
                )}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default ScenarioSelector;