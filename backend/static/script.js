// script.js - real-time dashboard za prikaz kvarova

let faultChart = null;
let ws = null;   
let pendingMessages = [];
let isChartReady = false;

// Pokreni sve komponente kad se stranica učita
document.addEventListener('DOMContentLoaded', () => {
    initEmptyChart();   // 1. Graf
    initWebSocket();     // 2. WebSocket  
    loadInitial();       // 3. Povijest (async)
});

// Inicijalizira prazan graf
function initEmptyChart() {
    const ctx = document.getElementById("faultChart");
    
    faultChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],  // Vremenske oznake 
            datasets: [{
                label: 'Abnormal vrijednost',
                data: [],
                pointRadius: 8,
                pointBackgroundColor: [],
                pointBorderColor: [],
                pointBorderWidth: 2,
                showLine: false,  // Ne spajaj točke linijom
                fill: false
            }]
        },
        options: {
            responsive: true,           
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    max: 1,
                    title: {
                        display: true,
                        text: 'Abnormal (0-1)'
                    }
                },
                x: {
                    title: {
                        display: true,
                        text: 'Vrijeme'
                    }
                }
            },
            plugins: {
                legend: {
                    display: true
                }
            }
        }
    });

    isChartReady = true;  
}

// WebSocket konekcija prema Go backendu
function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

    ws.onopen = () => {
        console.log("WebSocket connected to Go backend");
        flushPending();  // Procesuiranje buffered poruka
    };
    ws.onmessage = (evt) => {
        try {
            const msg = JSON.parse(evt.data);
            handleIncoming(msg);
        } catch (err) {
            console.error("Invalid WebSocket JSON:", evt.data, "Error:", err);
        }
    };
    ws.onclose = (event) => {
        console.warn(`WebSocket closed (code: ${event.code}), reconnecting...`);
        setTimeout(initWebSocket, 1000);
    };

    ws.onerror = (err) => {
        console.error("WebSocket connection error:", err);
    };
}

function handleIncoming(msg) {
    if (!isChartReady) {
        pendingMessages.push(msg);
        console.log(`Message buffered (${pendingMessages.length} in queue)`);
        return;
    }

    // Proslijeđivanje poruke odgovarajućem handleru
    if (msg.type === 'NEW_FAULT') {
        addFault(msg.data);     
        appendToChart(msg.data); 
    } else if (msg.type === 'STATUS') {
        updateStatus(msg.data); 
    }
}

// Procesuiranje buffered poruka (nakon što je chart spreman)
function flushPending() {
    while (pendingMessages.length > 0) {
        handleIncoming(pendingMessages.shift());
    }
}

// Učitavanje početnih podataka iz REST API-ja (nakon što je real-time channel postavljen)
async function loadInitial() {
    try {
        const res = await fetch('/api/faults');
        const data = await res.json();
        if (!Array.isArray(data)) {
            console.log("No historical data or invalid response:", data);
            return;
        }
        // 1. HTML lista (najnovije gore)
        data.reverse().forEach(addFault);
        // 2. graf (kronološki)
        data.forEach(appendToChart);
        // 3. buffered WS poruke
        flushPending();
    } catch (err) {
        console.error("Failed to load historical data:", err);
    }
}

    // Dodavanje nove točke u graf
function appendToChart(e) {
    if (!faultChart) return;
    const color = e.conclusion === 'KVAR' ? 'red' : 'green';
    faultChart.data.labels.push(new Date(e.timestamp).toLocaleTimeString());
    faultChart.data.datasets[0].data.push(e.abnormal);
    faultChart.data.datasets[0].pointBackgroundColor.push(color);
    faultChart.data.datasets[0].pointBorderColor.push(color);
    if (faultChart.data.labels.length > 50) {
        faultChart.data.labels.shift();
        faultChart.data.datasets[0].data.shift();
        faultChart.data.datasets[0].pointBackgroundColor.shift();
        faultChart.data.datasets[0].pointBorderColor.shift();
    }
    requestAnimationFrame(() => faultChart.update());
}

function addFault(e) {
    const container = document.getElementById("sensor-data");
    const color = e.conclusion === 'KVAR' ? 'red' : 'green';
    const faultEntry = `
        <div style="background:#ffe6e6; padding:6px; margin-bottom:6px; border-left:4px solid ${color}">
            <b>${new Date(e.timestamp).toLocaleString()}</b><br>
            Abnormal: ${e.abnormal}<br>
            Normal: ${e.normal}<br>
            Status: <span style="color:${color}; font-weight:bold">${e.conclusion}</span>
        </div>
    `;
    container.innerHTML = faultEntry + container.innerHTML;
}

// Ažuriranje statusa MQTT-a i baze
function updateStatus(d) {
    const mqttEl = document.getElementById("mqtt-status");
    const dbEl = document.getElementById("db-status");
    mqttEl.textContent = d.mqtt;
    mqttEl.style.color = d.mqtt === 'CONNECTED' ? 'green' : 'red';
    dbEl.textContent = d.db;
    dbEl.style.color = d.db === 'OK' ? 'green' : 'red';
}