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
        // Fetch history for the chart and dropdown
        const historyResponse = await fetch('/api/public/analytics/history');
        if (!historyResponse.ok) throw new Error("Failed to fetch history");
        const historyData = await historyResponse.json();

        populateDropdown(historyData);
        renderChart(historyData);

        // Fetch latest analytics for the cards
        await fetchAnalytics("");

        document.getElementById("loader").classList.add("hidden");
        document.getElementById("content").classList.remove("hidden");
        document.getElementById("date-selector").classList.remove("hidden");
    } catch (error) {
        console.error("Dashboard initialization failed:", error);
        document.getElementById("loader").innerHTML = `
            <i class="fa-solid fa-triangle-exclamation text-error fa-3x"></i>
            <p>Failed to load analytics data. Please try again later.</p>
        `;
    }
}

async function fetchAnalytics(date) {
    try {
        let url = '/api/public/analytics';
        // Add query param if viewing a specific past date
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

    // Reverse so newest is at the top of the dropdown after "Latest"
    const reversed = [...historyData].reverse();

    reversed.forEach(opt => {
        const dateObj = new Date(opt.date);
        // We know the date_key format in DB ends with _optimization. 
        // We need the YYYY-MM-DD prefix.
        const isoParts = opt.date.split("T")[0]; // YYYY-MM-DD

        const displayStr = dateObj.toLocaleDateString("en-US", { month: 'short', day: 'numeric', year: 'numeric' });

        const option = document.createElement("option");
        option.value = isoParts;
        option.textContent = `${displayStr} (Win Rate: ${(opt.success_rate * 100).toFixed(0)}%)`;
        selector.appendChild(option);
    });
}

function renderChart(historyData) {
    const ctx = document.getElementById('accuracyChart').getContext('2d');

    // Prepare data
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
                label: 'Prediction Success Rate (%)',
                data: dataPoints,
                borderColor: '#06b6d4', // accent color
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
                            return context.parsed.y + '% Success';
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
    // Top Stats
    if (optimization) {
        // Date formatting
        const dateObj = new Date(optimization.date);
        const dateStr = dateObj.toLocaleDateString("en-US", { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' });
        document.getElementById("analytics-date").textContent = `Insights from ${dateStr}`;

        // Numbers
        document.getElementById("stat-success-rate").textContent = `${(optimization.success_rate * 100).toFixed(1)}%`;

        // Color code success rate
        const srEl = document.getElementById("stat-success-rate");
        if (optimization.success_rate >= 0.7) srEl.className = "text-success";
        else if (optimization.success_rate < 0.5) srEl.className = "text-error";
        else srEl.className = "text-primary";

        document.getElementById("stat-wins").textContent = optimization.wins;
        document.getElementById("stat-losses").textContent = optimization.losses;
        document.getElementById("stat-total").textContent = optimization.total_bets;

        // Written Insights
        document.getElementById("ai-insights").innerHTML =
            `<strong>Overview:</strong> ${optimization.insights}<br><br>
            <strong>Winning Markets:</strong> ${optimization.winning_markets.join(", ") || "None"}<br>
            <strong>Losing Markets:</strong> ${optimization.losing_markets.join(", ") || "None"}`;

        document.getElementById("ai-improvements").textContent = optimization.prompt_improvements;
    } else {
        document.getElementById("analytics-date").textContent = "No optimization data available for this date.";
        document.getElementById("stat-success-rate").textContent = "0%";
        document.getElementById("stat-wins").textContent = "0";
        document.getElementById("stat-losses").textContent = "0";
        document.getElementById("stat-total").textContent = "0";
        document.getElementById("ai-insights").textContent = "Analysis pending...";
        document.getElementById("ai-improvements").textContent = "Analysis pending...";
    }

    // Historical Slips Grid
    if (slip && slip.recommendations && slip.recommendations.length > 0) {
        const container = document.getElementById("slips-container");
        container.innerHTML = ""; // Clear loader

        slip.recommendations.forEach(rec => {
            const card = document.createElement("div");
            card.className = `bet-card`;

            card.innerHTML = `
                <div class="bet-confidence">${rec.confidence}</div>
                <div class="bet-league">${rec.game.league} • ${rec.game.kickoff_time}</div>
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
