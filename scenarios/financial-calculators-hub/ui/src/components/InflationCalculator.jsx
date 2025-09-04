import React, { useState } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Download } from 'lucide-react';

const API_PORT = process.env.API_PORT || '20100';

function InflationCalculator() {
  const [inputs, setInputs] = useState({
    amount: 100000,
    years: 20,
    inflation_rate: 3
  });

  const [results, setResults] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setInputs({ ...inputs, [name]: parseFloat(value) || 0 });
  };

  const calculate = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`http://localhost:${API_PORT}/api/v1/calculate/inflation`, {
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
      'Metric,Value',
      `Current Amount,$${inputs.amount}`,
      `Years,${inputs.years}`,
      `Inflation Rate,${inputs.inflation_rate}%`,
      `Future Value Required,$${results.future_value.toFixed(2)}`,
      `Purchasing Power,$${results.purchasing_power.toFixed(2)}`,
      `Total Inflation,${results.total_inflation.toFixed(2)}%`
    ].join('\n');
    
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'inflation-calculation.csv';
    a.click();
  };

  const exportPDF = () => {
    const content = `
Inflation Impact Report
=======================
Generated: ${new Date().toLocaleString()}

INPUT PARAMETERS
Current Amount: $${inputs.amount.toLocaleString()}
Time Period: ${inputs.years} years
Inflation Rate: ${inputs.inflation_rate}%

RESULTS
Future Value Required: $${results?.future_value?.toFixed(2).toLocaleString()}
(Amount needed in ${inputs.years} years to maintain same purchasing power)

Current Purchasing Power in Future: $${results?.purchasing_power?.toFixed(2).toLocaleString()}
(What $${inputs.amount.toLocaleString()} today will be worth in ${inputs.years} years)

Total Inflation Impact: ${results?.total_inflation?.toFixed(1)}%

ANALYSIS
To maintain the same standard of living, you would need
$${results?.future_value?.toFixed(0).toLocaleString()} in ${inputs.years} years to have the 
equivalent purchasing power of $${inputs.amount.toLocaleString()} today.

Alternatively, if you have $${inputs.amount.toLocaleString()} in ${inputs.years} years,
it would only buy what $${results?.purchasing_power?.toFixed(0).toLocaleString()} buys today.
    `;
    
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'inflation-report.txt';
    a.click();
  };

  const getChartData = () => {
    if (!results) return [];
    
    const data = [];
    for (let year = 0; year <= inputs.years; year++) {
      const inflationFactor = Math.pow(1 + inputs.inflation_rate / 100, year);
      data.push({
        year,
        futureValue: Math.round(inputs.amount * inflationFactor),
        purchasingPower: Math.round(inputs.amount / inflationFactor)
      });
    }
    return data;
  };

  return (
    <div>
      <h2>Inflation Calculator</h2>
      
      <div className="form-row">
        <div className="form-group">
          <label>
            Current Amount ($)
          </label>
          <input
            type="number"
            name="amount"
            value={inputs.amount}
            onChange={handleInputChange}
            min="0"
          />
        </div>
        
        <div className="form-group">
          <label>
            Time Period (Years)
          </label>
          <input
            type="number"
            name="years"
            value={inputs.years}
            onChange={handleInputChange}
            min="1"
            max="50"
          />
        </div>
        
        <div className="form-group">
          <label>
            Annual Inflation Rate (%)
            <span className="label-hint"> (3% typical)</span>
          </label>
          <input
            type="number"
            name="inflation_rate"
            value={inputs.inflation_rate}
            onChange={handleInputChange}
            min="0"
            max="20"
            step="0.1"
          />
        </div>
      </div>
      
      <div className="button-group">
        <button className="btn-primary" onClick={calculate} disabled={loading}>
          {loading ? <span className="loading"></span> : 'Calculate Inflation Impact'}
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
            <h3>Inflation Impact Analysis</h3>
            <div className="result-item">
              <span className="result-label">Future Value Required</span>
              <span className="result-value">
                ${Math.round(results.future_value).toLocaleString()}
              </span>
            </div>
            <div className="result-item">
              <span className="result-label">
                To maintain the purchasing power of ${inputs.amount.toLocaleString()} today
              </span>
            </div>
            
            <div style={{ borderTop: '1px solid var(--border)', paddingTop: '0.75rem', marginTop: '0.75rem' }}>
              <div className="result-item">
                <span className="result-label">Current Amount's Future Purchasing Power</span>
                <span className="result-value" style={{ color: '#ef4444' }}>
                  ${Math.round(results.purchasing_power).toLocaleString()}
                </span>
              </div>
              <div className="result-item">
                <span className="result-label">
                  What ${inputs.amount.toLocaleString()} in {inputs.years} years will actually buy
                </span>
              </div>
            </div>
            
            <div style={{ borderTop: '1px solid var(--border)', paddingTop: '0.75rem', marginTop: '0.75rem' }}>
              <div className="result-item">
                <span className="result-label">Total Inflation Over Period</span>
                <span className="result-value">
                  {results.total_inflation.toFixed(1)}%
                </span>
              </div>
              <div className="result-item">
                <span className="result-label">Average Annual Impact</span>
                <span className="result-value">
                  -{((1 - results.purchasing_power / inputs.amount) * 100 / inputs.years).toFixed(2)}% purchasing power/year
                </span>
              </div>
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
          
          {getChartData().length > 0 && (
            <div className="chart-container">
              <h3>Inflation Impact Over Time</h3>
              <ResponsiveContainer width="100%" height={300}>
                <LineChart data={getChartData()}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="year" label={{ value: 'Years', position: 'insideBottom', offset: -5 }} />
                  <YAxis tickFormatter={(value) => `$${(value / 1000).toFixed(0)}k`} />
                  <Tooltip formatter={(value) => `$${value.toLocaleString()}`} />
                  <Legend />
                  <Line 
                    type="monotone" 
                    dataKey="futureValue" 
                    stroke="#10b981" 
                    strokeWidth={2}
                    name="Future Value Needed"
                    dot={false}
                  />
                  <Line 
                    type="monotone" 
                    dataKey="purchasingPower" 
                    stroke="#ef4444" 
                    strokeWidth={2}
                    name="Purchasing Power"
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

export default InflationCalculator;