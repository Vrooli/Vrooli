import React, { useState } from 'react';
import { Calculator, TrendingUp, Home, DollarSign } from 'lucide-react';
import FIRECalculator from './components/FIRECalculator';
import CompoundInterestCalculator from './components/CompoundInterestCalculator';
import MortgageCalculator from './components/MortgageCalculator';
import InflationCalculator from './components/InflationCalculator';

function App() {
  const [activeTab, setActiveTab] = useState('fire');

  const tabs = [
    { id: 'fire', name: 'FIRE', icon: TrendingUp },
    { id: 'compound', name: 'Compound Interest', icon: Calculator },
    { id: 'mortgage', name: 'Mortgage', icon: Home },
    { id: 'inflation', name: 'Inflation', icon: DollarSign },
  ];

  return (
    <div>
      <div className="header">
        <div className="container">
          <h1>💰 Financial Calculators Hub</h1>
          <p>Professional financial planning tools for retirement, investments, and budgeting</p>
        </div>
      </div>

      <div className="container">
        <div className="tabs">
          {tabs.map(tab => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                className={`tab ${activeTab === tab.id ? 'active' : ''}`}
                onClick={() => setActiveTab(tab.id)}
              >
                <Icon size={20} style={{ marginRight: '0.5rem', verticalAlign: 'middle' }} />
                {tab.name}
              </button>
            );
          })}
        </div>

        <div className="calculator-card">
          {activeTab === 'fire' && <FIRECalculator />}
          {activeTab === 'compound' && <CompoundInterestCalculator />}
          {activeTab === 'mortgage' && <MortgageCalculator />}
          {activeTab === 'inflation' && <InflationCalculator />}
        </div>
      </div>
    </div>
  );
}

export default App;