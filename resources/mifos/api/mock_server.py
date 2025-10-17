#!/usr/bin/env python3
"""
Mock Mifos Server - Simple API implementation for microfinance operations
"""

from flask import Flask, jsonify, request
import json
import os
from datetime import datetime
import uuid

app = Flask(__name__)
app.config['PORT'] = int(os.environ.get('MIFOS_PORT', 8030))

# In-memory data storage
clients = {}
loans = {}
savings_accounts = {}

# Health check endpoint
@app.route('/fineract-provider/api/v1/offices', methods=['GET'])
def health_check():
    """Health check endpoint that mimics real Fineract"""
    return jsonify({
        "offices": [
            {"id": 1, "name": "Head Office", "status": "active"}
        ],
        "status": "healthy",
        "timestamp": datetime.now().isoformat()
    })

# Authentication endpoint
@app.route('/fineract-provider/api/v1/authentication', methods=['POST'])
def authenticate():
    """Mock authentication endpoint"""
    auth_data = request.get_json() or {}
    username = auth_data.get('username', 'mifos')
    
    token = str(uuid.uuid4())
    return jsonify({
        "authenticated": True,
        "username": username,
        "token": token,
        "permissions": ["ALL_FUNCTIONS"]
    })

# Client management
@app.route('/fineract-provider/api/v1/clients', methods=['GET'])
def get_clients():
    """List all clients"""
    return jsonify({
        "totalFilteredRecords": len(clients),
        "pageItems": list(clients.values())
    })

@app.route('/fineract-provider/api/v1/clients', methods=['POST'])
def create_client():
    """Create a new client"""
    client_data = request.get_json()
    client_id = str(uuid.uuid4())
    
    client = {
        "id": client_id,
        "accountNo": f"CL{len(clients)+1:06d}",
        "firstname": client_data.get('firstname', 'John'),
        "lastname": client_data.get('lastname', 'Doe'),
        "status": "active",
        "activationDate": datetime.now().isoformat(),
        "officeId": 1,
        "savingsAccountId": None,
        "externalId": client_data.get('externalId')
    }
    
    clients[client_id] = client
    return jsonify(client), 201

@app.route('/fineract-provider/api/v1/clients/<client_id>', methods=['GET'])
def get_client(client_id):
    """Get client by ID"""
    if client_id in clients:
        return jsonify(clients[client_id])
    return jsonify({"error": "Client not found"}), 404

# Loan operations
@app.route('/fineract-provider/api/v1/loans', methods=['GET'])
def get_loans():
    """List all loans"""
    return jsonify({
        "totalFilteredRecords": len(loans),
        "pageItems": list(loans.values())
    })

@app.route('/fineract-provider/api/v1/loans', methods=['POST'])
def create_loan():
    """Create a new loan"""
    loan_data = request.get_json()
    loan_id = str(uuid.uuid4())
    
    loan = {
        "id": loan_id,
        "accountNo": f"LN{len(loans)+1:06d}",
        "clientId": loan_data.get('clientId'),
        "loanProductId": loan_data.get('loanProductId', 1),
        "principal": loan_data.get('principal', 10000),
        "loanTermFrequency": loan_data.get('loanTermFrequency', 12),
        "interestRatePerPeriod": loan_data.get('interestRatePerPeriod', 2.5),
        "status": {
            "id": 300,
            "code": "loanStatusType.active",
            "value": "Active"
        },
        "disbursementDate": datetime.now().isoformat(),
        "currency": {
            "code": loan_data.get('currency', 'USD'),
            "name": "US Dollar",
            "decimalPlaces": 2
        }
    }
    
    loans[loan_id] = loan
    return jsonify(loan), 201

@app.route('/fineract-provider/api/v1/loans/<loan_id>', methods=['GET'])
def get_loan(loan_id):
    """Get loan by ID"""
    if loan_id in loans:
        return jsonify(loans[loan_id])
    return jsonify({"error": "Loan not found"}), 404

@app.route('/fineract-provider/api/v1/loans/<loan_id>/transactions', methods=['POST'])
def loan_transaction(loan_id):
    """Process loan transaction (payment)"""
    if loan_id not in loans:
        return jsonify({"error": "Loan not found"}), 404
    
    transaction_data = request.get_json()
    transaction = {
        "id": str(uuid.uuid4()),
        "loanId": loan_id,
        "type": transaction_data.get('type', 'repayment'),
        "amount": transaction_data.get('amount'),
        "date": datetime.now().isoformat(),
        "status": "completed"
    }
    
    return jsonify(transaction), 201

# Savings accounts
@app.route('/fineract-provider/api/v1/savingsaccounts', methods=['GET'])
def get_savings_accounts():
    """List all savings accounts"""
    return jsonify({
        "totalFilteredRecords": len(savings_accounts),
        "pageItems": list(savings_accounts.values())
    })

@app.route('/fineract-provider/api/v1/savingsaccounts', methods=['POST'])
def create_savings_account():
    """Create a new savings account"""
    account_data = request.get_json()
    account_id = str(uuid.uuid4())
    
    account = {
        "id": account_id,
        "accountNo": f"SA{len(savings_accounts)+1:06d}",
        "clientId": account_data.get('clientId'),
        "savingsProductId": account_data.get('savingsProductId', 1),
        "accountBalance": 0,
        "status": "active",
        "activatedOnDate": datetime.now().isoformat(),
        "currency": {
            "code": account_data.get('currency', 'USD'),
            "name": "US Dollar",
            "decimalPlaces": 2
        }
    }
    
    savings_accounts[account_id] = account
    
    # Link to client if provided
    if account_data.get('clientId') in clients:
        clients[account_data['clientId']]['savingsAccountId'] = account_id
    
    return jsonify(account), 201

# Loan products
@app.route('/fineract-provider/api/v1/loanproducts', methods=['GET'])
def get_loan_products():
    """List available loan products"""
    return jsonify([
        {
            "id": 1,
            "name": "Microenterprise Loan",
            "shortName": "MEL",
            "principal": 10000,
            "minPrincipal": 1000,
            "maxPrincipal": 50000,
            "interestRatePerPeriod": 2.5,
            "currency": {"code": "USD"}
        },
        {
            "id": 2,
            "name": "Agricultural Loan",
            "shortName": "AGL",
            "principal": 5000,
            "minPrincipal": 500,
            "maxPrincipal": 25000,
            "interestRatePerPeriod": 2.0,
            "currency": {"code": "USD"}
        }
    ])

# Savings products
@app.route('/fineract-provider/api/v1/savingsproducts', methods=['GET'])
def get_savings_products():
    """List available savings products"""
    return jsonify([
        {
            "id": 1,
            "name": "Basic Savings",
            "shortName": "BSA",
            "nominalAnnualInterestRate": 1.5,
            "currency": {"code": "USD"}
        },
        {
            "id": 2,
            "name": "High Yield Savings",
            "shortName": "HYS",
            "nominalAnnualInterestRate": 3.0,
            "currency": {"code": "USD"}
        }
    ])

# Root endpoint
@app.route('/', methods=['GET'])
def root():
    """Root endpoint"""
    return jsonify({
        "service": "Mifos X Mock Server",
        "version": "1.0.0",
        "status": "running"
    })

# Seed demo data
def seed_demo_data():
    """Create initial demo data"""
    # Create demo clients
    for i in range(3):
        client_id = str(uuid.uuid4())
        clients[client_id] = {
            "id": client_id,
            "accountNo": f"CL{i+1:06d}",
            "firstname": ["John", "Jane", "Bob"][i],
            "lastname": ["Doe", "Smith", "Johnson"][i],
            "status": "active",
            "activationDate": datetime.now().isoformat(),
            "officeId": 1,
            "savingsAccountId": None
        }
        
        # Create savings account for each client
        account_id = str(uuid.uuid4())
        savings_accounts[account_id] = {
            "id": account_id,
            "accountNo": f"SA{i+1:06d}",
            "clientId": client_id,
            "savingsProductId": 1,
            "accountBalance": 1000 * (i + 1),
            "status": "active",
            "activatedOnDate": datetime.now().isoformat(),
            "currency": {"code": "USD", "name": "US Dollar", "decimalPlaces": 2}
        }
        clients[client_id]['savingsAccountId'] = account_id
        
        # Create a loan for first two clients
        if i < 2:
            loan_id = str(uuid.uuid4())
            loans[loan_id] = {
                "id": loan_id,
                "accountNo": f"LN{i+1:06d}",
                "clientId": client_id,
                "loanProductId": 1,
                "principal": 5000 * (i + 1),
                "loanTermFrequency": 12,
                "interestRatePerPeriod": 2.5,
                "status": {
                    "id": 300,
                    "code": "loanStatusType.active",
                    "value": "Active"
                },
                "disbursementDate": datetime.now().isoformat(),
                "currency": {"code": "USD", "name": "US Dollar", "decimalPlaces": 2}
            }

if __name__ == '__main__':
    seed_demo_data()
    app.run(host='0.0.0.0', port=app.config['PORT'], debug=False)