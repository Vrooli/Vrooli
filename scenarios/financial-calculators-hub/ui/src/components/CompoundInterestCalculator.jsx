import React, { useState } from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Download } from 'lucide-react';

function CompoundInterestCalculator() {
  const [inputs, setInputs] = useState({
    principal: 10000,
    annual_rate: 7,
    years: 10,
    monthly_contribution: 500,
    compound_frequency: 'monthly'
  });

  const [results, setResults] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setInputs({ ...inputs, [name]: name === 'compound_frequency' ? value : parseFloat(value) });
  };

  const calculate = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch('/api/v1/calculate/compound-interest', {
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
      'Year,Balance,Contribution,Interest',
      ...results.year_by_year.map(y => 
        `${y.year},${y.balance.toFixed(2)},${y.contribution.toFixed(2)},${y.interest.toFixed(2)}`
      )
    ].join('\n');
    
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'compound-interest.csv';
    a.click();
  };

  const exportPDF = () => {
    const content = `
Compound Interest Calculation Report
====================================
Generated: ${new Date().toLocaleString()}

INPUT PARAMETERS
Principal: $${inputs.principal.toLocaleString()}
Annual Rate: ${inputs.annual_rate}%
Years: ${inputs.years}
Monthly Contribution: $${inputs.monthly_contribution.toLocaleString()}
Compound Frequency: ${inputs.compound_frequency}

RESULTS
Final Amount: $${results?.final_amount?.toFixed(2).toLocaleString()}
Total Contributions: $${results?.total_contributions?.toFixed(2).toLocaleString()}
Total Interest: $${results?.total_interest?.toFixed(2).toLocaleString()}

YEAR BY YEAR BREAKDOWN
${results?.year_by_year?.map(y => 
  `Year ${y.year}: Balance: $${y.balance.toFixed(2).toLocaleString()}`
).join('\n')}
    `;
    
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'compound-interest.txt';
    a.click();
  };

  return (
    <div>
      <h2>Compound Interest Calculator</h2>
      
      <div className="form-row">
        <div className="form-group">
          <label>
            Initial Principal ($)
          </label>
          <input
            type="number"
            name="principal"
            value={inputs.principal}
            onChange={handleInputChange}
            min="0"
          />
        </div>
        
        <div className="form-group">
          <label>
            Annual Interest Rate (%)
          </label>
          <input
            type="number"
            name="annual_rate"
            value={inputs.annual_rate}
            onChange={handleInputChange}
            min="0"
            max="100"
            step="0.1"
          />
        </div>
        
        <div className="form-group">
          <label>
            Investment Period (Years)
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
      </div>
      
      <div className="form-row">
        <div className="form-group">
          <label>
            Monthly Contribution ($)
            <span className="label-hint"> (optional)</span>
          </label>
          <input
            type="number"
            name="monthly_contribution"
            value={inputs.monthly_contribution}
            onChange={handleInputChange}
            min="0"
          />
        </div>
        
        <div className="form-group">
          <label>
            Compound Frequency
          </label>
          <select
            name="compound_frequency"
            value={inputs.compound_frequency}
            onChange={handleInputChange}
          >
            <option value="monthly">Monthly</option>
            <option value="quarterly">Quarterly</option>
            <option value="annually">Annually</option>
          </select>
        </div>
      </div>
      
      <div className="button-group">
        <button className="btn-primary" onClick={calculate} disabled={loading}>
          {loading ? <span className="loading"></span> : 'Calculate Growth'}
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
            <h3>Investment Growth Summary</h3>
            <div className="result-item">
              <span className="result-label">Final Amount</span>
              <span className="result-value">${Math.round(results.final_amount).toLocaleString()}</span>
            </div>
            <div className="result-item">
              <span className="result-label">Total Contributions</span>
              <span className="result-value">${Math.round(results.total_contributions).toLocaleString()}</span>
            </div>
            <div className="result-item">
              <span className="result-label">Total Interest Earned</span>
              <span className="result-value" style={{ color: '#10b981' }}>
                ${Math.round(results.total_interest).toLocaleString()}
              </span>
            </div>
            <div className="result-item">
              <span className="result-label">Return on Investment</span>
              <span className="result-value">
                {((results.total_interest / results.total_contributions) * 100).toFixed(1)}%
              </span>
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
          
          {results.year_by_year && results.year_by_year.length > 0 && (
            <div className="chart-container">
              <h3>Year-by-Year Growth</h3>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={results.year_by_year}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="year" />
                  <YAxis tickFormatter={(value) => `$${(value / 1000).toFixed(0)}k`} />
                  <Tooltip formatter={(value) => `$${Math.round(value).toLocaleString()}`} />
                  <Legend />
                  <Bar dataKey="contribution" stackId="a" fill="#3b82f6" name="Contributions" />
                  <Bar dataKey="interest" stackId="a" fill="#10b981" name="Interest" />
                </BarChart>
              </ResponsiveContainer>
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default CompoundInterestCalculator;