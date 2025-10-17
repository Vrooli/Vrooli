import React, { useState } from 'react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { Download } from 'lucide-react';

function MortgageCalculator() {
  const [inputs, setInputs] = useState({
    loan_amount: 300000,
    annual_rate: 4.5,
    years: 30,
    down_payment: 60000,
    extra_monthly_payment: 0
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
      const response = await fetch('/api/v1/calculate/mortgage', {
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
      'Month,Payment,Principal,Interest,Remaining Balance',
      ...results.amortization_schedule.slice(0, 360).map(p => 
        `${p.month},${p.payment.toFixed(2)},${p.principal.toFixed(2)},${p.interest.toFixed(2)},${p.remaining_balance.toFixed(2)}`
      )
    ].join('\n');
    
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'mortgage-amortization.csv';
    a.click();
  };

  const exportPDF = () => {
    const content = `
Mortgage Calculation Report
===========================
Generated: ${new Date().toLocaleString()}

LOAN PARAMETERS
Loan Amount: $${inputs.loan_amount.toLocaleString()}
Down Payment: $${inputs.down_payment.toLocaleString()}
Principal: $${(inputs.loan_amount - inputs.down_payment).toLocaleString()}
Annual Rate: ${inputs.annual_rate}%
Loan Term: ${inputs.years} years
Extra Monthly Payment: $${inputs.extra_monthly_payment.toLocaleString()}

RESULTS
Monthly Payment: $${results?.monthly_payment?.toFixed(2).toLocaleString()}
Total Interest: $${results?.total_interest?.toFixed(2).toLocaleString()}
Total Paid: $${results?.total_paid?.toFixed(2).toLocaleString()}
Payoff Date: ${results?.payoff_date}

Interest Savings from Extra Payments: $${
  inputs.extra_monthly_payment > 0 
    ? ((inputs.loan_amount - inputs.down_payment) * inputs.annual_rate / 100 * inputs.years - results?.total_interest)?.toFixed(2).toLocaleString()
    : '0'
}
    `;
    
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'mortgage-calculation.txt';
    a.click();
  };

  const getChartData = () => {
    if (!results?.amortization_schedule) return [];
    
    const yearlyData = [];
    for (let i = 0; i < results.amortization_schedule.length; i += 12) {
      const yearPayments = results.amortization_schedule.slice(i, i + 12);
      if (yearPayments.length === 0) break;
      
      const totalPrincipal = yearPayments.reduce((sum, p) => sum + p.principal, 0);
      const totalInterest = yearPayments.reduce((sum, p) => sum + p.interest, 0);
      
      yearlyData.push({
        year: Math.floor(i / 12) + 1,
        principal: Math.round(totalPrincipal),
        interest: Math.round(totalInterest),
        balance: Math.round(yearPayments[yearPayments.length - 1].remaining_balance)
      });
    }
    
    return yearlyData;
  };

  return (
    <div>
      <h2>Mortgage Calculator</h2>
      
      <div className="form-row">
        <div className="form-group">
          <label>
            Home Price ($)
          </label>
          <input
            type="number"
            name="loan_amount"
            value={inputs.loan_amount}
            onChange={handleInputChange}
            min="0"
          />
        </div>
        
        <div className="form-group">
          <label>
            Down Payment ($)
            <span className="label-hint"> ({((inputs.down_payment / inputs.loan_amount) * 100).toFixed(1)}%)</span>
          </label>
          <input
            type="number"
            name="down_payment"
            value={inputs.down_payment}
            onChange={handleInputChange}
            min="0"
            max={inputs.loan_amount}
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
            max="20"
            step="0.1"
          />
        </div>
      </div>
      
      <div className="form-row">
        <div className="form-group">
          <label>
            Loan Term (Years)
          </label>
          <input
            type="number"
            name="years"
            value={inputs.years}
            onChange={handleInputChange}
            min="1"
            max="40"
          />
        </div>
        
        <div className="form-group">
          <label>
            Extra Monthly Payment ($)
            <span className="label-hint"> (optional)</span>
          </label>
          <input
            type="number"
            name="extra_monthly_payment"
            value={inputs.extra_monthly_payment}
            onChange={handleInputChange}
            min="0"
          />
        </div>
      </div>
      
      <div className="button-group">
        <button className="btn-primary" onClick={calculate} disabled={loading}>
          {loading ? <span className="loading"></span> : 'Calculate Mortgage'}
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
            <h3>Mortgage Summary</h3>
            <div className="result-item">
              <span className="result-label">Loan Amount</span>
              <span className="result-value">
                ${(inputs.loan_amount - inputs.down_payment).toLocaleString()}
              </span>
            </div>
            <div className="result-item">
              <span className="result-label">Monthly Payment (P&I)</span>
              <span className="result-value">${Math.round(results.monthly_payment).toLocaleString()}</span>
            </div>
            <div className="result-item">
              <span className="result-label">Total Interest Paid</span>
              <span className="result-value" style={{ color: '#ef4444' }}>
                ${Math.round(results.total_interest).toLocaleString()}
              </span>
            </div>
            <div className="result-item">
              <span className="result-label">Total Amount Paid</span>
              <span className="result-value">${Math.round(results.total_paid).toLocaleString()}</span>
            </div>
            <div className="result-item">
              <span className="result-label">Payoff Date</span>
              <span className="result-value">{new Date(results.payoff_date).toLocaleDateString()}</span>
            </div>
            
            <div className="export-buttons">
              <button className="btn-secondary" onClick={exportCSV}>
                <Download size={16} style={{ marginRight: '0.5rem' }} />
                Export Amortization Schedule
              </button>
              <button className="btn-secondary" onClick={exportPDF}>
                <Download size={16} style={{ marginRight: '0.5rem' }} />
                Export Report
              </button>
            </div>
          </div>
          
          {getChartData().length > 0 && (
            <div className="chart-container">
              <h3>Principal vs Interest by Year</h3>
              <ResponsiveContainer width="100%" height={300}>
                <AreaChart data={getChartData()}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="year" label={{ value: 'Year', position: 'insideBottom', offset: -5 }} />
                  <YAxis tickFormatter={(value) => `$${(value / 1000).toFixed(0)}k`} />
                  <Tooltip formatter={(value) => `$${value.toLocaleString()}`} />
                  <Area type="monotone" dataKey="principal" stackId="1" stroke="#3b82f6" fill="#3b82f6" name="Principal" />
                  <Area type="monotone" dataKey="interest" stackId="1" stroke="#ef4444" fill="#ef4444" name="Interest" />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default MortgageCalculator;