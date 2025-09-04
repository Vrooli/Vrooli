import React, { useState } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Download } from 'lucide-react';

const API_PORT = process.env.API_PORT || '20100';

function FIRECalculator() {
  const [inputs, setInputs] = useState({
    current_age: 30,
    current_savings: 100000,
    annual_income: 100000,
    annual_expenses: 50000,
    savings_rate: 50,
    expected_return: 7,
    target_withdrawal_rate: 4
  });

  const [results, setResults] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    const newInputs = { ...inputs, [name]: parseFloat(value) };
    
    if (name === 'annual_income' || name === 'annual_expenses') {
      const income = name === 'annual_income' ? parseFloat(value) : inputs.annual_income;
      const expenses = name === 'annual_expenses' ? parseFloat(value) : inputs.annual_expenses;
      if (income > 0 && expenses > 0) {
        newInputs.savings_rate = ((income - expenses) / income) * 100;
      }
    }
    
    setInputs(newInputs);
  };

  const calculate = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`http://localhost:${API_PORT}/api/v1/calculate/fire`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(inputs)
      });
      
      if (!response.ok) {
        throw new Error('Calculation failed');
      }
      
      const data = await response.json();
      setResults(data);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const exportCSV = () => {
    if (!results) return;
    
    const csv = [
      'Age,Projected Nest Egg',
      ...Object.entries(results.projected_nest_egg_by_age || {})
        .map(([age, value]) => `${age},${value.toFixed(2)}`)
    ].join('\n');
    
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'fire-calculation.csv';
    a.click();
  };

  const exportPDF = () => {
    const content = `
Financial Independence Calculation Report
=========================================
Generated: ${new Date().toLocaleString()}

INPUT PARAMETERS
Current Age: ${inputs.current_age}
Current Savings: $${inputs.current_savings.toLocaleString()}
Annual Income: $${inputs.annual_income.toLocaleString()}
Annual Expenses: $${inputs.annual_expenses.toLocaleString()}
Savings Rate: ${inputs.savings_rate.toFixed(1)}%
Expected Return: ${inputs.expected_return}%
Withdrawal Rate: ${inputs.target_withdrawal_rate}%

RESULTS
Retirement Age: ${results?.retirement_age?.toFixed(1)}
Years to Retirement: ${results?.years_to_retirement?.toFixed(1)}
Target Nest Egg: $${results?.target_nest_egg?.toLocaleString()}
Monthly Savings Required: $${results?.monthly_savings_required?.toFixed(0).toLocaleString()}
    `;
    
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'fire-calculation.txt';
    a.click();
  };

  const formatChartData = () => {
    if (!results?.projected_nest_egg_by_age) return [];
    
    return Object.entries(results.projected_nest_egg_by_age)
      .map(([age, value]) => ({
        age: parseInt(age),
        value: Math.round(value)
      }))
      .sort((a, b) => a.age - b.age);
  };

  return (
    <div>
      <h2>FIRE (Financial Independence Retire Early) Calculator</h2>
      
      <div className="form-row">
        <div className="form-group">
          <label>
            Current Age
          </label>
          <input
            type="number"
            name="current_age"
            value={inputs.current_age}
            onChange={handleInputChange}
            min="18"
            max="100"
          />
        </div>
        
        <div className="form-group">
          <label>
            Current Savings ($)
          </label>
          <input
            type="number"
            name="current_savings"
            value={inputs.current_savings}
            onChange={handleInputChange}
            min="0"
          />
        </div>
        
        <div className="form-group">
          <label>
            Annual Income ($)
          </label>
          <input
            type="number"
            name="annual_income"
            value={inputs.annual_income}
            onChange={handleInputChange}
            min="0"
          />
        </div>
      </div>
      
      <div className="form-row">
        <div className="form-group">
          <label>
            Annual Expenses ($)
          </label>
          <input
            type="number"
            name="annual_expenses"
            value={inputs.annual_expenses}
            onChange={handleInputChange}
            min="0"
          />
        </div>
        
        <div className="form-group">
          <label>
            Savings Rate (%)
            <span className="label-hint"> (auto-calculated)</span>
          </label>
          <input
            type="number"
            name="savings_rate"
            value={inputs.savings_rate.toFixed(1)}
            onChange={handleInputChange}
            min="0"
            max="100"
          />
        </div>
        
        <div className="form-group">
          <label>
            Expected Annual Return (%)
          </label>
          <input
            type="number"
            name="expected_return"
            value={inputs.expected_return}
            onChange={handleInputChange}
            min="0"
            max="30"
            step="0.1"
          />
        </div>
      </div>
      
      <div className="form-row">
        <div className="form-group">
          <label>
            Target Withdrawal Rate (%)
            <span className="label-hint"> (4% rule typical)</span>
          </label>
          <input
            type="number"
            name="target_withdrawal_rate"
            value={inputs.target_withdrawal_rate}
            onChange={handleInputChange}
            min="1"
            max="10"
            step="0.1"
          />
        </div>
      </div>
      
      <div className="button-group">
        <button className="btn-primary" onClick={calculate} disabled={loading}>
          {loading ? <span className="loading"></span> : 'Calculate FIRE'}
        </button>
      </div>
      
      {error && (
        <div className="error">
          Error: {error}
        </div>
      )}
      
      {results && (
        <>
          <div className="results">
            <h3>Your Path to Financial Independence</h3>
            <div className="result-item">
              <span className="result-label">Retirement Age</span>
              <span className="result-value">{results.retirement_age.toFixed(1)} years</span>
            </div>
            <div className="result-item">
              <span className="result-label">Years to Retirement</span>
              <span className="result-value">{results.years_to_retirement.toFixed(1)} years</span>
            </div>
            <div className="result-item">
              <span className="result-label">Target Nest Egg</span>
              <span className="result-value">${Math.round(results.target_nest_egg).toLocaleString()}</span>
            </div>
            <div className="result-item">
              <span className="result-label">Monthly Savings Required</span>
              <span className="result-value">${Math.round(results.monthly_savings_required).toLocaleString()}</span>
            </div>
            
            <div className="export-buttons">
              <button className="btn-secondary" onClick={exportCSV}>
                <Download size={16} style={{ marginRight: '0.5rem' }} />
                Export CSV
              </button>
              <button className="btn-secondary" onClick={exportPDF}>
                <Download size={16} style={{ marginRight: '0.5rem' }} />
                Export Report
              </button>
            </div>
          </div>
          
          {formatChartData().length > 0 && (
            <div className="chart-container">
              <h3>Projected Nest Egg Growth</h3>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={formatChartData()}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="age" label={{ value: 'Age', position: 'insideBottom', offset: -5 }} />
                  <YAxis 
                    label={{ value: 'Nest Egg ($)', angle: -90, position: 'insideLeft' }}
                    tickFormatter={(value) => `$${(value / 1000).toFixed(0)}k`}
                  />
                  <Tooltip 
                    formatter={(value) => `$${value.toLocaleString()}`}
                    labelFormatter={(label) => `Age ${label}`}
                  />
                  <Legend />
                  <Line 
                    type="monotone" 
                    dataKey="value" 
                    stroke="#2563eb" 
                    strokeWidth={2}
                    name="Portfolio Value"
                    dot={false}
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default FIRECalculator;