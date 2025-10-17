function startDemoTelemetry(wss) {
    console.log('Starting demo telemetry streams...');
    
    // Satellite telemetry
    setInterval(() => {
        const satelliteData = {
            stream: 'satellite_position',
            timestamp: Date.now(),
            value: Math.sin(Date.now() / 10000) * 90,  // Latitude
            data: {
                latitude: Math.sin(Date.now() / 10000) * 90,
                longitude: Math.cos(Date.now() / 10000) * 180,
                altitude: 400 + Math.random() * 10,
                velocity: 7.8 + Math.random() * 0.1
            }
        };
        
        broadcast(wss, satelliteData);
    }, 1000);
    
    // Sensor network
    setInterval(() => {
        const sensorData = {
            stream: 'sensor_network',
            timestamp: Date.now(),
            value: 20 + Math.random() * 10,  // Temperature
            data: {
                temperature: 20 + Math.random() * 10,
                humidity: 40 + Math.random() * 20,
                pressure: 1013 + Math.random() * 10
            }
        };
        
        broadcast(wss, sensorData);
    }, 2000);
    
    // System metrics
    setInterval(() => {
        const metricsData = {
            stream: 'system_metrics',
            timestamp: Date.now(),
            value: Math.random() * 100,  // CPU usage
            data: {
                cpu: Math.random() * 100,
                memory: Math.random() * 100,
                disk: Math.random() * 100,
                network: Math.random() * 1000
            }
        };
        
        broadcast(wss, metricsData);
    }, 3000);
}

function broadcast(wss, data) {
    wss.clients.forEach(client => {
        if (client.readyState === 1) {  // WebSocket.OPEN
            client.send(JSON.stringify(data));
        }
    });
}

module.exports = { startDemoTelemetry };
