#!/usr/bin/env node
// Test Geth JSON-RPC connection

const http = require('http');

// Test JSON-RPC connection
const testConnection = () => {
    const data = JSON.stringify({
        jsonrpc: '2.0',
        method: 'eth_blockNumber',
        params: [],
        id: 1
    });

    const options = {
        hostname: 'localhost',
        port: 8545,
        path: '/',
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Content-Length': data.length
        }
    };

    const req = http.request(options, (res) => {
        let responseData = '';
        
        res.on('data', (chunk) => {
            responseData += chunk;
        });
        
        res.on('end', () => {
            try {
                const response = JSON.parse(responseData);
                const blockNumber = parseInt(response.result, 16);
                console.log('Connected to Geth!');
                console.log(`Current block number: ${blockNumber}`);
                process.exit(0);
            } catch (error) {
                console.error('Failed to parse response:', error);
                process.exit(1);
            }
        });
    });

    req.on('error', (error) => {
        console.error('Connection failed:', error.message);
        process.exit(1);
    });

    req.write(data);
    req.end();
};

// Run test
console.log('Testing Geth connection...');
testConnection();