let myChart = null;

document.addEventListener("DOMContentLoaded", () => {
    initDashboard();

    // Listen for dropdown changes
    document.getElementById("date-selector").addEventListener("change", (e) => {
        const selectedDate = e.target.value;
        fetchAnalytics(selectedDate);
    });
});

async function initDashboard() {
    try {
        // Fetch history for the chart, dropdown, and all-time stats
        const historyResponse = await fetch('/api/public/analytics/history');
        if (!historyResponse.ok) throw new Error("Failed to fetch history");
        const historyData = await historyResponse.json();

        calculateAllTimeStats(historyData);
        populateDropdown(historyData);
        renderChart(historyData);

        // Fetch latest analytics for the daily dive
        await fetchAnalytics("");

        document.getElementById("loader").classList.add("hidden");
        document.getElementById("content").classList.remove("hidden");
    } catch (error) {
        console.error("Dashboard initialization failed:", error);
        document.getElementById("loader").innerHTML = `
            <i class="fa-solid fa-triangle-exclamation text-error fa-3x"></i>
            <p>Failed to load analytics data. Please try again later.</p>
        `;
    }
}

function calculateAllTimeStats(historyData) {
    if (!historyData || historyData.length === 0) return;

    let totalWins = 0;
    let totalLosses = 0;
    let totalBets = 0;

    const winningMarkets = {};
    const losingMarkets = {};

    historyData.forEach(opt => {
        totalWins += opt.wins || 0;
        totalLosses += opt.losses || 0;
        totalBets += opt.total_bets || 0;

        if (opt.winning_markets) {
            opt.winning_markets.forEach(m => {
                winningMarkets[m] = (winningMarkets[m] || 0) + 1;
            });
        }
        if (opt.losing_markets) {
            opt.losing_markets.forEach(m => {
                losingMarkets[m] = (losingMarkets[m] || 0) + 1;
            });
        }
    });

    // Populate Top Cards
    const accuracy = totalBets > 0 ? ((totalWins / totalBets) * 100).toFixed(1) : 0;
    const accEl = document.getElementById("all-time-accuracy");
    accEl.textContent = `${accuracy}%`;
    if (accuracy >= 65) accEl.className = "text-success";
    else if (accuracy < 50) accEl.className = "text-error";
    else accEl.className = "text-primary";

    document.getElementById("all-time-wins").textContent = totalWins;
    document.getElementById("all-time-total").textContent = totalBets;

    // Recent Form (Last 5 days accuracy trend)
    const recent = historyData.slice(-5);
    const recentWins = recent.reduce((sum, o) => sum + (o.wins || 0), 0);
    const recentTotal = recent.reduce((sum, o) => sum + (o.total_bets || 0), 0);
    const recentAcc = recentTotal > 0 ? ((recentWins / recentTotal) * 100).toFixed(0) : 0;
    document.getElementById("recent-form").textContent = `${recentAcc}%`;

    // Populate Markets
    const sortMarkets = (marketObj) => {
        return Object.entries(marketObj).sort((a, b) => b[1] - a[1]);
    };

    const bestList = document.getElementById("best-markets-list");
    bestList.innerHTML = "";
    const sortedBest = sortMarkets(winningMarkets).slice(0, 3);
    if(sortedBest.length === 0) bestList.innerHTML = "<li>No data yet</li>";
    sortedBest.forEach(([market, count]) => {
        bestList.innerHTML += `<li><span>${market}</span> <span class="market-count">${count}x</span></li>`;
    });

    const worstList = document.getElementById("worst-markets-list");
    worstList.innerHTML = "";
    const sortedWorst = sortMarkets(losingMarkets).slice(0, 3);
    if(sortedWorst.length === 0) worstList.innerHTML = "<li>No data yet</li>";
    sortedWorst.forEach(([market, count]) => {
        worstList.innerHTML += `<li><span>${market}</span> <span class="market-count">${count}x</span></li>`;
    });
}

async function fetchAnalytics(date) {
    try {
        let url = '/api/public/analytics';
        if (date) {
            url += `?date=${encodeURIComponent(date)}`;
        }

        const response = await fetch(url);
        if (!response.ok) throw new Error("Failed to fetch analytics");
        const data = await response.json();

        if (data.status !== "success") throw new Error("API returned failure");

        renderAnalytics(data.optimization, data.slip);
    } catch (error) {
        console.error("Failed to fetch specific analytics:", error);
    }
}

function populateDropdown(historyData) {
    const selector = document.getElementById("date-selector");
    selector.innerHTML = `<option value="">Latest Optimization</option>`; // Reset dropdown first for multiple calls
    const reversed = [...historyData].reverse();

    reversed.forEach(opt => {
        const dateObj = new Date(opt.date);
        const isoParts = opt.date.split("T")[0]; // YYYY-MM-DD
        const displayStr = dateObj.toLocaleDateString("en-US", { month: 'short', day: 'numeric', year: 'numeric' });

        const option = document.createElement("option");
        option.value = isoParts;
        option.textContent = `${displayStr} (Acc: ${(opt.success_rate * 100).toFixed(0)}%)`;
        selector.appendChild(option);
    });
}

function renderChart(historyData) {
    const ctx = document.getElementById('accuracyChart').getContext('2d');

    const labels = historyData.map(opt => {
        return new Date(opt.date).toLocaleDateString("en-US", { month: 'short', day: 'numeric' });
    });

    const dataPoints = historyData.map(opt => (opt.success_rate * 100).toFixed(1));

    if (myChart) {
        myChart.destroy();
    }

    myChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: 'Daily Accuracy (%)',
                data: dataPoints,
                borderColor: '#06b6d4',
                backgroundColor: 'rgba(6, 182, 212, 0.1)',
                borderWidth: 3,
                pointBackgroundColor: '#3b82f6',
                pointBorderColor: '#fff',
                pointHoverBackgroundColor: '#fff',
                pointHoverBorderColor: '#3b82f6',
                pointRadius: 5,
                pointHoverRadius: 7,
                fill: true,
                tension: 0.4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    labels: {
                        color: '#e2e8f0',
                        font: { family: "'Outfit', sans-serif", size: 14 }
                    }
                },
                tooltip: {
                    backgroundColor: 'rgba(15, 23, 42, 0.9)',
                    titleColor: '#fff',
                    bodyColor: '#cbd5e1',
                    borderColor: 'rgba(255, 255, 255, 0.1)',
                    borderWidth: 1,
                    padding: 12,
                    displayColors: false,
                    callbacks: {
                        label: function (context) {
                            return context.parsed.y + '% Accuracy';
                        }
                    }
                }
            },
            scales: {
                y: {
                    min: 0,
                    max: 100,
                    grid: {
                        color: 'rgba(255,255,255,0.05)',
                        drawBorder: false
                    },
                    ticks: {
                        color: '#94a3b8',
                        callback: function (value) { return value + "%" }
                    }
                },
                x: {
                    grid: {
                        color: 'rgba(255,255,255,0.05)',
                        drawBorder: false
                    },
                    ticks: {
                        color: '#94a3b8'
                    }
                }
            }
        }
    });
}

function renderAnalytics(optimization, slip) {
    if (optimization) {
        const accuracy = (optimization.success_rate * 100).toFixed(1);
        
        const accEl = document.getElementById("daily-accuracy");
        accEl.textContent = `${accuracy}%`;
        
        if (accuracy >= 65) accEl.className = "text-success";
        else if (accuracy < 50) accEl.className = "text-error";
        else accEl.className = "text-primary";

        document.getElementById("daily-record").textContent = `${optimization.wins}W / ${optimization.losses}L`;
        document.getElementById("ai-insights").textContent = optimization.insights;
        document.getElementById("ai-improvements").textContent = optimization.prompt_improvements;
    } else {
        document.getElementById("daily-accuracy").textContent = "0%";
        document.getElementById("daily-record").textContent = "0 / 0";
        document.getElementById("ai-insights").textContent = "Analysis pending...";
        document.getElementById("ai-improvements").textContent = "Analysis pending...";
    }

    if (slip && slip.recommendations && slip.recommendations.length > 0) {
        const container = document.getElementById("slips-container");
        container.innerHTML = ""; 

        slip.recommendations.forEach(rec => {
            const card = document.createElement("div");
            card.className = `bet-card`;

            card.innerHTML = `
                <div class="bet-confidence">${rec.confidence}</div>
                <div class="bet-league">${rec.game.league}</div>
                <div class="bet-teams">${rec.game.home_team} vs ${rec.game.away_team}</div>
                
                <div class="bet-reasoning">${rec.reasoning}</div>
                
                <div class="bet-details">
                    <span class="bet-selection">${rec.selection} (${rec.market})</span>
                    <span class="bet-odds">@ ${rec.odds.toFixed(2)}</span>
                </div>
            `;
            container.appendChild(card);
        });
    } else {
        document.getElementById("slips-container").innerHTML = `
            <p style="color: #94a3b8; grid-column: 1 / -1; text-align: center;">No prediction data found for this analysis period.</p>
        `;
    }
}
