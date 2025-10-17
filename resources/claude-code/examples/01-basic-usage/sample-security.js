const express = require('express');
const app = express();

app.get('/user/:id', (req, res) => {
    const userId = req.params.id;
    const query = `SELECT * FROM users WHERE id = ${userId}`;
    // Execute query (SQL injection vulnerable)
    res.json({ query: query });
});
